#!/bin/sh
# Download the potion-base-8M model files into ./model.
# config.json + tokenizer.json are committed; only the ~29 MB weights are gitignored,
# but this fetches all three so a fresh checkout is self-sufficient.
set -e
DIR="$(dirname "$0")/model"
BASE="https://huggingface.co/minishlab/potion-base-8M/resolve/main"
mkdir -p "$DIR"
for f in config.json tokenizer.json model.safetensors; do
  echo "fetching $f"
  curl -fsSL -o "$DIR/$f" "$BASE/$f"
done
echo "done -> $DIR"
