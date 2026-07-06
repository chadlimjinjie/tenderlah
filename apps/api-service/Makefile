.PHONY: dev build sqlc swag

dev:
	go run ./cmd/server

build:
	go build -o bin/api-server ./cmd/server

# Regenerate type-safe DB queries (requires: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest)
sqlc:
	sqlc generate

# Regenerate Swagger docs (requires: go install github.com/swaggo/swag/cmd/swag@latest)
swag:
	swag init -g cmd/server/main.go -o docs
