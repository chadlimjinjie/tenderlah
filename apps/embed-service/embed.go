package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// Model is a loaded model2vec static embedder: a vocab tokenizer plus a
// row-major [Rows][Dim] float32 embedding matrix. "Inference" is a row
// lookup + mean-pool + optional L2 normalize — no neural forward pass.
type Model struct {
	Name      string
	Dim       int
	Rows      int
	emb       []float32 // Rows*Dim, row-major
	tok       *Tokenizer
	normalize bool
}

type modelConfig struct {
	HiddenDim int  `json:"hidden_dim"`
	Normalize bool `json:"normalize"`
}

func LoadModel(dir string) (*Model, error) {
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read config.json: %w", err)
	}
	var cfg modelConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}
	tok, err := loadTokenizer(filepath.Join(dir, "tokenizer.json"))
	if err != nil {
		return nil, err
	}
	rows, dim, emb, err := loadEmbeddings(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		return nil, err
	}
	return &Model{
		Name:      "potion-base-8M",
		Dim:       dim,
		Rows:      rows,
		emb:       emb,
		tok:       tok,
		normalize: cfg.Normalize,
	}, nil
}

// Embed returns the L2-normalized mean of the token embedding rows.
// Empty input (no tokens) returns a zero vector.
func (m *Model) Embed(text string) []float32 {
	ids := m.tok.Encode(text)
	out := make([]float32, m.Dim)
	if len(ids) == 0 {
		return out
	}
	for _, id := range ids {
		row := m.emb[id*m.Dim : (id+1)*m.Dim]
		for j, v := range row {
			out[j] += v
		}
	}
	inv := float32(1) / float32(len(ids))
	for j := range out {
		out[j] *= inv
	}
	if m.normalize {
		var norm float64
		for _, v := range out {
			norm += float64(v) * float64(v)
		}
		if norm > 0 {
			n := float32(math.Sqrt(norm))
			for j := range out {
				out[j] /= n
			}
		}
	}
	return out
}
