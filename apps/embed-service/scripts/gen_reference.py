#!/usr/bin/env python3
"""Regenerate reference_vectors.json from the official model2vec lib.

The Go test (embed_test.go) asserts our pure-Go tokenizer reproduces these
vectors, so this is the source of truth for tokenizer fidelity. Run only when
the model changes.

    pip install model2vec   # numpy + tokenizers, no torch
    python scripts/gen_reference.py
"""
import json
from model2vec import StaticModel

TEXTS = ["the cat sat", "a cat is sitting", "stock market crash", "Hello, World!"]

m = StaticModel.from_pretrained("minishlab/potion-base-8M")
V = m.encode(TEXTS)
out = {
    "model": "potion-base-8M",
    "dim": int(V.shape[1]),
    "cases": [
        {"text": t, "vec": [round(float(x), 6) for x in V[i]]}
        for i, t in enumerate(TEXTS)
    ],
}
with open("reference_vectors.json", "w") as f:
    json.dump(out, f)
print(f"wrote reference_vectors.json ({len(TEXTS)} cases, dim {out['dim']})")
