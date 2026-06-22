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
		return "", fmt.Errorf("deepseek chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("deepseek returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
