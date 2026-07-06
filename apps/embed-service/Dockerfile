# --- fetch weights (gitignored, so pull at build time) ---
FROM alpine:3.20 AS weights
RUN apk add --no-cache wget
WORKDIR /w
RUN wget -q https://huggingface.co/minishlab/potion-base-8M/resolve/main/model.safetensors

# --- build static binary ---
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /embed-service .

# --- runtime ---
FROM alpine:3.20
WORKDIR /app
COPY --from=build /embed-service /app/embed-service
COPY model/config.json model/tokenizer.json /app/model/
COPY --from=weights /w/model.safetensors /app/model/model.safetensors
# GOMEMLIMIT keeps the GC inside the 0.5 GB box (matrix ~30 MB + runtime).
ENV PORT=4107 MODEL_DIR=/app/model GOMEMLIMIT=400MiB
EXPOSE 4107
ENTRYPOINT ["/app/embed-service"]
