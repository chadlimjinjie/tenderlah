package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// loadEmbeddings reads the "embeddings" F32 tensor from a safetensors file.
// Format: uint64 little-endian header length, JSON header, then raw tensor
// bytes. See https://github.com/huggingface/safetensors for the spec.
func loadEmbeddings(path string) (rows, dim int, data []float32, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("read safetensors: %w", err)
	}
	if len(raw) < 8 {
		return 0, 0, nil, fmt.Errorf("safetensors too short")
	}
	hlen := binary.LittleEndian.Uint64(raw[:8])
	if 8+hlen > uint64(len(raw)) {
		return 0, 0, nil, fmt.Errorf("safetensors header out of range")
	}
	var hdr map[string]struct {
		Dtype       string   `json:"dtype"`
		Shape       []int    `json:"shape"`
		DataOffsets []uint64 `json:"data_offsets"`
	}
	if err := json.Unmarshal(raw[8:8+hlen], &hdr); err != nil {
		return 0, 0, nil, fmt.Errorf("parse safetensors header: %w", err)
	}
	t, ok := hdr["embeddings"]
	if !ok {
		return 0, 0, nil, fmt.Errorf("no 'embeddings' tensor in safetensors")
	}
	if t.Dtype != "F32" {
		return 0, 0, nil, fmt.Errorf("expected F32 embeddings, got %s", t.Dtype)
	}
	if len(t.Shape) != 2 || len(t.DataOffsets) != 2 {
		return 0, 0, nil, fmt.Errorf("unexpected embeddings shape/offsets: %v %v", t.Shape, t.DataOffsets)
	}
	rows, dim = t.Shape[0], t.Shape[1]
	base := 8 + hlen
	start, end := base+t.DataOffsets[0], base+t.DataOffsets[1]
	if end > uint64(len(raw)) || uint64(rows*dim*4) != end-start {
		return 0, 0, nil, fmt.Errorf("embeddings data size mismatch")
	}
	buf := raw[start:end]
	data = make([]float32, rows*dim)
	for i := range data {
		data[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return rows, dim, data, nil
}
