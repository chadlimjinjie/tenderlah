# api-service-rust

Rust (axum) port of `apps/api-service` (Go/Gin) — the Midtone API. Strict
behavior-compatible rewrite: same routes, JSON shapes (verified byte-for-byte
against the Go service on live data), status codes, auth semantics, and env
var names. The consumer is the external `apps/web-beta` Next.js frontend. The
Go service has been removed; its `docs/swagger.json` contract lives in git history.

## Toolchain

Requires **Rust ≥ 1.85** (the locked dependency tree uses edition2024 crates;
older Cargo fails parsing e.g. `time-macros`' manifest). `rust-toolchain.toml`
pins 1.96.1 — the version this port was built and verified with — which
rustup-based deploy buildpacks install automatically.

## Run

```
cp .env.example .env
make dev                      # or: cargo run
make build                    # or: cargo build --release  (bin: api-service-rust)
make test                     # or: cargo test
```

Defaults: `PORT` 4003 in code (`.env.example` sets 4002, same as the Go
service's mismatch, preserved deliberately).

## Layout

| Rust | ports (Go) |
|---|---|
| `src/main.rs` | `cmd/server/main.go` |
| `src/auth.rs`, `src/middleware.rs` | `internal/auth/jwks.go`, `internal/middleware/auth.go` |
| `src/db/*` | `internal/db/*` (sqlc queries copied verbatim; dynamic search via `QueryBuilder`) |
| `src/llm.rs`, `src/storage.rs`, `src/db/embed.rs` | `internal/llm`, `internal/storage`, `internal/db/embed.go` |
| `src/gebiz.rs` | `internal/gebizdetail/parse.go` (+ its test fixture) |
| `src/handlers/*` | `internal/handlers/*` (prompts copied verbatim) |

## Notes / gaps vs the Go service

- **Swagger UI (`/api/docs`) not ported** — returns 404. The Go service's
  generated `docs/swagger.json` is in git history.
- **PgBouncer**: `DATABASE_URL` points at Supabase's transaction-mode pooler,
  so every query runs with `.persistent(false)` (unnamed prepared statements —
  the sqlx equivalent of pgx's `QueryExecModeCacheDescribe`). Don't remove
  this, or you'll get "prepared statement already exists" errors.
- **Timestamps** serialize in Go's exact RFC3339Nano form (server-local
  offset, trailing-zero-trimmed fractional seconds) via
  `db::models::serialize_go_time`. Postgres `date` columns serialize as
  midnight UTC (`...T00:00:00Z`), matching pgx.
- **400 validation texts differ**: Go leaks go-playground/validator messages;
  this port returns clean messages (same status, same `{"error": ...}` shape).
- **500 bodies** echo the Rust error string (Go echoes the Go error string —
  same shape, different text).
- **405 vs 404**: axum returns 405 for a wrong method on a known path; Gin
  returned 404. Gin also redirected trailing slashes; axum does not.
- **JSON string escaping**: Go escapes `&`, `<`, `>` as `&` etc.; serde
  does not. Semantically identical JSON.
- **In-flight guards** (brief/timeline/match/scrape) are per-process
  `DashSet`s, same as the Go `sync.Map`s — swap for pg advisory locks if this
  ever runs with multiple replicas.
- The unused Go `RequireOrg` middleware and the always-nil `tenderRow.match`
  field are carried over as documented no-ops.
