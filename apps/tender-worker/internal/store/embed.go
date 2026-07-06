package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// embedDim is the fixed output size of Embed, matching all-minilm's
// native dimensionality. The tender.embedding column type must match.
const embedDim = 384

const embedModel = "all-minilm"

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed calls Ollama's native embeddings endpoint to turn text into a
// 384-dim vector. No retries — a single failed call is the caller's problem
// (e.g. log and proceed with a NULL embedding).
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
