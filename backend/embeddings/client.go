package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client calls the local embeddings service (Flask + fastembed, all-MiniLM-L6-v2, 384-dim).
// Replaces the previous OpenRouter-backed client.
type Client struct {
	url        string
	httpClient *http.Client
}

func NewClient() *Client {
	url := os.Getenv("EMBEDDINGS_URL")
	if url == "" {
		url = "http://embeddings:7100/embeddings"
	}
	return &Client{
		url:        url,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type embedRequest struct {
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Embed generates embeddings for a batch of text chunks.
// Returns one 384-dim float64 slice per input, in input order.
func (c *Client) Embed(inputs []string) ([][]float64, error) {
	return c.EmbedCtx(context.Background(), inputs)
}

// EmbedCtx is the context-aware variant. Passing the worker's per-document
// deadline context lets a hung embeddings service be interrupted instead of
// blocking until the HTTP client's own 60s timeout (audit item H11).
func (c *Client) EmbedCtx(ctx context.Context, inputs []string) ([][]float64, error) {
	bodyBytes, err := json.Marshal(embedRequest{Input: inputs})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embeddings service returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var result embedResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		embeddings[d.Index] = d.Embedding
	}

	return embeddings, nil
}
