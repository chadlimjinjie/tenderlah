package main

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// TestEmbedMatchesReference is the tokenizer-drift guard: reference_vectors.json
// holds vectors produced offline by the official Python model2vec lib. If our
// pure-Go normalize/pre-tokenize/wordpiece pipeline diverges, cosine drops.
// Regenerate the reference with scripts/gen_reference.py (see model/README.md).
func TestEmbedMatchesReference(t *testing.T) {
	m, err := LoadModel("./model")
	if err != nil {
		t.Skipf("model not present (run ./fetch-model.sh): %v", err)
	}
	b, err := os.ReadFile("reference_vectors.json")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}
	var ref struct {
		Dim   int `json:"dim"`
		Cases []struct {
			Text string    `json:"text"`
			Vec  []float32 `json:"vec"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		t.Fatal(err)
	}
	if m.Dim != ref.Dim {
		t.Fatalf("dim mismatch: got %d want %d", m.Dim, ref.Dim)
	}
	for _, c := range ref.Cases {
		got := m.Embed(c.Text)
		if cos := cosine(got, c.Vec); cos < 0.9999 {
			t.Errorf("text %q: cosine %.6f < 0.9999 (tokenizer drift?)", c.Text, cos)
		}
	}
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
