package ai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const defaultModel = "deepseek-chat"

// Usage captures the token accounting the model provider returns for a single
// chat completion. The openai-go SDK surfaces this as resp.Usage on every
// non-streamed ChatCompletion response. We persist it per call so the
// developer tools can compute spend (tokens × per-million price).
type Usage struct {
	PromptTokens       int64
	CompletionTokens   int64
	TotalTokens        int64
	CachedPromptTokens int64 // prompt tokens served from the provider's cache (cheaper)
	ReasoningTokens    int64 // chain-of-thought tokens (deepseek-reasoner / o1-style)
}

// LLMClient wraps the OpenAI-compatible client for DeepSeek chat completions.
type LLMClient struct {
	client *openai.Client
	model  string
}

// NewLLMClient creates a client pre-configured for DeepSeek.
//
// It reads these environment variables:
//   - DEEPSEEK_API_KEY (required) — fallback is OPENAI_API_KEY
//   - DEEPSEEK_BASE_URL (optional, defaults to https://api.deepseek.com)
//   - MODEL (optional, defaults to deepseek-chat)
//
// You can also set OPENAI_API_KEY and OPENAI_BASE_URL for a generic setup.
func NewLLMClient() *LLMClient {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		log.Println("[llm] WARNING: neither DEEPSEEK_API_KEY nor OPENAI_API_KEY set — AI calls will fail")
	}

	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	model := os.Getenv("MODEL")
	if model == "" {
		model = defaultModel
	}

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
		// Allow up to 5 minutes for a response (DeepSeek can be slow on free tier)
		option.WithHTTPClient(&http.Client{Timeout: 5 * time.Minute}),
	)

	return &LLMClient{
		client: &client,
		model:  model,
	}
}

// Chat sends a system + user prompt to DeepSeek (Chat Completions) and returns
// the response text. This preserves the existing interface so all callers
// (jobs.go, coach/handler.go, ai/handler.go) continue working unchanged.
func (c *LLMClient) Chat(systemPrompt, userPrompt string) (string, error) {
	text, _, err := c.ChatWithUsage(systemPrompt, userPrompt)
	return text, err
}

// ChatWithUsage is the same call as Chat but also returns the provider's token
// accounting for the request. Use this from any code path that needs to track
// spend (developer tooling, mock course generation, analytics). The token
// counts come straight off the SDK's resp.Usage object — no estimation.
func (c *LLMClient) ChatWithUsage(systemPrompt, userPrompt string) (string, Usage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Model: openai.ChatModel(c.model),
	})
	if err != nil {
		return "", Usage{}, fmt.Errorf("deepseek chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("deepseek returned no choices")
	}

	u := Usage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	// Cached prompt tokens are surfaced inside PromptTokensDetails when the
	// provider serves part of the prompt from its prefix cache (DeepSeek does
	// this automatically for repeated system prompts). They're billed at a
	// discount, so we keep them separate from the headline prompt count.
	if ptd := resp.Usage.PromptTokensDetails; ptd.CachedTokens != 0 {
		u.CachedPromptTokens = ptd.CachedTokens
	}
	if ctd := resp.Usage.CompletionTokensDetails; ctd.ReasoningTokens != 0 {
		u.ReasoningTokens = ctd.ReasoningTokens
	}

	return resp.Choices[0].Message.Content, u, nil
}

// Model returns the model id this client is configured to call. Used by the
// token-usage logger so each persisted row records what it was billed as.
func (c *LLMClient) Model() string { return c.model }
