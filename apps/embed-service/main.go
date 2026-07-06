package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var model *Model

func main() {
	dir := getenv("MODEL_DIR", "./model")
	m, err := LoadModel(dir)
	if err != nil {
		log.Fatalf("load model: %v", err)
	}
	model = m
	log.Printf("loaded %s: %d tokens, dim %d", m.Name, m.Rows, m.Dim)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/embed/health", handleHealth)
	mux.HandleFunc("/api/embed", handleEmbed)

	port := getenv("PORT", "4107")
	log.Printf("embed-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "model": model.Name, "dim": model.Dim,
	})
}

type embedRequest struct {
	Texts []string `json:"texts"`
}

type embedResponse struct {
	Vectors [][]float32 `json:"vectors"`
	Dim     int         `json:"dim"`
}

func handleEmbed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req embedRequest
	// 10 MB cap: a static-embedding batch has no reason to be larger.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Texts) == 0 {
		writeErr(w, http.StatusBadRequest, "texts must be a non-empty array")
		return
	}
	vecs := make([][]float32, len(req.Texts))
	for i, t := range req.Texts {
		vecs[i] = model.Embed(t)
	}
	writeJSON(w, http.StatusOK, embedResponse{Vectors: vecs, Dim: model.Dim})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
