# GeBIZ compliance guardrails

GeBIZ's terms of use (`/rss-terms-of-use.html`, `/terms-of-use.html`) name **DSTA**
as the authority and restrict how GeBIZ data may be linked, reproduced, and reused.
TenderLah touches GeBIZ in three places, each gated by a flag. **Do not enable the
detail-scrape or deep-link flags until written DSTA permission is on file.**

| # | Touch-point | Flag | Default | Enable only when |
|---|-------------|------|---------|------------------|
| 1 | RSS ingestion (`internal/gebiz`) | `GEBIZ_ENABLED` (tender-worker) | off | DSTA permits — resume with `=true` |
| 2 | Listing / detail display (`api-server-go` tender queries) | `GEBIZ_LISTING_ENABLED` (api-server-go) | off | DSTA permits reproducing RSS fields — show with `=true` |
| 3 | Deep-link to opportunity page (`web-tenders` tender view) | `NEXT_PUBLIC_GEBIZ_LINKS_ENABLED` | `false` | DSTA permits hyperlinking (encode any conditions) |
| 4 | Opportunity-page detail scrape (`api-server-go` `internal/gebizdetail`) | `GEBIZ_DETAIL_SCRAPE_ENABLED` (api-server-go) | off | DSTA gives **written** permission to reproduce content |

GeBIZ is **ceased** by default: the worker does not ingest it (#1) and `api-server-go`
excludes `source = 'gebiz'` from every tender read + 404s GeBIZ deep-links (#2), so nothing
GeBIZ surfaces on `/tenders`. Existing rows stay in the DB (hidden), so re-enabling is a
config flip. Flip all relevant flags to `true` once DSTA permission is on file.

Permission-request letter draft: `docs/gebiz-dsta-permission-request.md`.

If DSTA denies #4, keep `GEBIZ_DETAIL_SCRAPE_ENABLED` off permanently and rely only
on the RSS fields (title, link, dates, calling entity, category).
