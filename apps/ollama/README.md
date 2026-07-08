# apps/ollama — local embedding model server

Stock `ollama/ollama` image that pulls `all-minilm` into its persistent volume
(`/root/.ollama`) on first boot, so `tender-worker` and `api-server-go` have a
real embedding backend to call. (Baking the model in at build time doesn't work
here — the persistent volume mounted at `/root/.ollama` shadows the image's
baked-in models, so the pull has to happen at runtime, into the volume.)
`all-minilm` (23M params, 384-dim) was picked over the larger
`nomic-embed-text` (137M params, 768-dim) to keep the container's CPU/RAM
footprint down — it's the heavier model that was actually running before.

Ollama's hosted cloud API (`https://ollama.com`, used elsewhere in this repo
for chat completions) does not serve embedding models — only large
chat/generation models. Embeddings need an actual Ollama server, hence this
service.

Port 11434, no auth (not exposed publicly — only `tender-worker` and
`api-server-go` talk to it, over the private network). Point
`OLLAMA_EMBED_BASE_URL` at this service's internal address, e.g.
`http://ollama.railway.internal:11434` (Railway) or `http://localhost:11434`
(local dev with `docker run -p 11434:11434 <image>`).

Add more embedding/chat models the same way — extra `RUN ollama pull <model>`
lines — if a future use case needs one.
