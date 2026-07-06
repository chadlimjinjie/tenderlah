package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const DefaultModel = "gpt-oss:120b-cloud"

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	http    *http.Client
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
	Stream bool `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func New(baseURL, apiKey, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		http:    &http.Client{},
	}
}

// Generate sends a chat completion and returns the assistant message content.
func (c *Client) Generate(ctx context.Context, system, prompt string) (string, error) {
	return c.generate(ctx, system, prompt, false)
}

// GenerateJSON requests a JSON-formatted response.
func (c *Client) GenerateJSON(ctx context.Context, system, prompt string) (string, error) {
	return c.generate(ctx, system, prompt, true)
}

func (c *Client) generate(ctx context.Context, system, prompt string, jsonMode bool) (string, error) {
	req := chatRequest{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}
	if jsonMode {
		req.ResponseFormat = &struct {
			Type string `json:"type"`
		}{Type: "json_object"}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama %d: %s", resp.StatusCode, string(b))
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return result.Choices[0].Message.Content, nil
}
