package store

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != embedModel {
			t.Fatalf("expected model %q, got %q", embedModel, req.Model)
		}
		if req.Input != "road resurfacing works" {
			t.Fatalf("unexpected input: %q", req.Input)
		}
		vec := make([]float32, embedDim)
		json.NewEncoder(w).Encode(embedResponse{Embeddings: [][]float32{vec}})
	}))
	defer srv.Close()

	vec, err := Embed(context.Background(), srv.URL, "", "road resurfacing works")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != embedDim {
		t.Fatalf("expected dim %d, got %d", embedDim, len(vec))
	}
}

func TestEmbedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := Embed(context.Background(), srv.URL, "", "text"); err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
}
