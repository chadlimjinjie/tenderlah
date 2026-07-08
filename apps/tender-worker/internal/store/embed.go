package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// embedDim is the fixed output size of Embed, matching embed-service's
// potion-base-8M dimensionality. The tender.embedding column type must match.
const embedDim = 256

type embedRequest struct {
	Texts []string `json:"texts"`
}

type embedResponse struct {
	Vectors [][]float32 `json:"vectors"`
	Dim     int         `json:"dim"`
}

// Embed calls the internal embed-service to turn text into a 256-dim vector.
// No retries — a single failed call is the caller's problem (e.g. log and
// proceed with a NULL embedding). apiKey is unused (embed-service has no auth)
// but kept for signature compatibility with the api-service copy.
func Embed(ctx context.Context, baseURL, apiKey, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{Texts: []string{text}})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed-service %d: %s", resp.StatusCode, string(b))
	}

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Vectors) == 0 {
		return nil, fmt.Errorf("embed-service: empty response")
	}
	return out.Vectors[0], nil
}
