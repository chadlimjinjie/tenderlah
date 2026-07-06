# embed-service (port 4107)

Tiny local text-embedding HTTP service for **near-duplicate detection / clustering**.
Fits in **0.5 GB RAM / 0.5 vCPU** because it runs a **model2vec static embedding**
model (`MinishLab/potion-base-8M`, 256-dim) — "inference" is a token lookup +
mean-pool + L2 normalize, so there is **no transformer, no PyTorch, no ONNX
Runtime**. Pure Go stdlib `net/http` plus `golang.org/x/text` (Unicode NFD only).

## Run

```bash
./fetch-model.sh          # downloads model/model.safetensors (~29 MB, gitignored)
go run .                  # PORT=4107, MODEL_DIR=./model
```

Docker (self-fetches weights, caps GC at 400 MiB):

```bash
docker build -t embed-service .
docker run --rm -p 4107:4107 --memory=512m --cpus=0.5 embed-service
```

## API

| Method | Path                  | Body / Response |
|--------|-----------------------|-----------------|
| GET    | `/api/embed/health`   | → `{"status":"ok","model":"potion-base-8M","dim":256}` |
| POST   | `/api/embed`          | `{"texts":["...","..."]}` → `{"vectors":[[...]],"dim":256}` |

Vectors are unit length, so dedup with a plain dot product (cosine). Near-dup
texts score ~0.85+; unrelated ~0. Empty text → zero vector.

```bash
curl -sX POST localhost:4107/api/embed -H 'Content-Type: application/json' \
  -d '{"texts":["the cat sat on the mat","a cat is sitting on a mat"]}'
```

## Config

`PORT` (default `4107`), `MODEL_DIR` (default `./model`).

## Tests

`go test ./...` — asserts the pure-Go tokenizer reproduces reference vectors from
the official Python model2vec lib (guards against tokenizer drift). Regenerate the
reference with `python scripts/gen_reference.py` if the model changes.

## Model

Only `embed-service` is the sole consumer of these files; not shared with other
Go modules (repo convention: copy small code, don't share modules). To upgrade
quality, swap to `potion-retrieval-32M` (512-dim, ~130 MB) — same code path,
larger matrix; update `fetch-model.sh` + `Dockerfile` URLs and regenerate the
reference vectors.
