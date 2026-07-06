package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const embedModel = "all-minilm"

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed calls Ollama's native embeddings endpoint to turn text into a
// 384-dim vector.
//
// This is independently written to mirror apps/tender-worker/internal/store/embed.go.
// tender-worker and api-server-go are separate Go modules, and this is a
// small, dependency-light function, so duplicating it beats introducing a
// shared module. Any change here must be mirrored there too — both outputs
// share the same vector space as the tender.embedding column, and a drift
// between the two would silently corrupt relevance ranking.
func Embed(ctx context.Context, baseURL, apiKey, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{Model: embedModel, Input: text})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed %d: %s", resp.StatusCode, string(b))
	}

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed: empty response")
	}
	return out.Embeddings[0], nil
}
