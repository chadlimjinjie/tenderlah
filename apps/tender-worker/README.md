# tender-worker

Pulls Singapore [GeBIZ](https://www.gebiz.gov.sg) tender-opportunity RSS feeds
and upserts each opportunity into Postgres (`gebiz_tender` table) for later
analysis. Runs once and exits — schedule repeats externally (cron, systemd
timer, etc).

## How it works

- On startup it ensures the `gebiz_tender` table exists (`CREATE TABLE IF NOT EXISTS`),
  pulls every configured feed once, then exits.
- Each item is keyed by its stable `OPPORTUNITY_ID` and written with an idempotent
  UPSERT (`ON CONFLICT (opportunity_id) DO UPDATE`), so restarts and repeated pulls
  never create duplicates. `first_seen_at` is preserved; `last_seen_at` and mutable
  fields (`status`, `closing_date`, …) are refreshed each pull.

## Env vars

| Var | Required | Default | Notes |
|---|---|---|---|
| `DATABASE_URL` | yes | — | Same Postgres as the rest of the repo |

The feed list is a hardcoded constant in `internal/gebiz/feeds.go` (`gebiz.Feeds`) —
add or remove category URLs there. GeBIZ publishes one feed per category, shaped
`https://www.gebiz.gov.sg/rss/{Category}-CREATE_BO_FEED.xml`; the category is
derived from the URL and stored on each row.

## Run

```bash
cd apps/tender-worker
go run ./cmd/server

go test ./...   # parser + dedup self-checks
```
