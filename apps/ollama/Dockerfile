FROM ollama/ollama:latest

EXPOSE 11434

# Serve in the foreground (PID 1, clean signals). A background job waits for the
# server to answer, then pulls all-minilm into the persistent volume at
# /root/.ollama — runtime, not build time, because the mounted volume shadows
# anything baked in. `ollama pull` needs the daemon up, so it can only run after
# serve starts; it's a no-op once the model is already in the volume.
ENTRYPOINT ["/bin/sh", "-c", "(until ollama list >/dev/null 2>&1; do sleep 1; done; ollama pull all-minilm) & exec ollama serve"]