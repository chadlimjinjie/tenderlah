# bizsafe-worker

One-shot importer for the WSH Council's [bizSAFE Self-Help](https://www.tal.sg/wshc/programmes/bizsafe/bizsafe-e-services)
Excel file. Scrapes the page for the current `.xlsx` download link, parses every
row of its hidden "Data" sheet, and upserts into Postgres (`bizsafe_companies` table).

## How it works

- Scrapes the bizSAFE Self-Help page for the `.xlsx` link (the filename changes per
  update, so it's discovered each run, not hardcoded).
- Downloads the file and reads the `Data` sheet: `UEN, COMPANY_NAME, CERT, TYPE,
  ACTIVE, EXPIRY, STATUS, APPROVAL, LEVEL, DESCRPTION`.
- Each row is keyed by its certificate number (`CERT`, e.g. `E00001`) and written
  with an idempotent `UPSERT (ON CONFLICT (cert) DO UPDATE)`, so re-running never
  creates duplicates. `first_seen_at` is preserved; `last_seen_at` and all other
  fields refresh on every run.

## Env vars

| Var | Required | Notes |
|---|---|---|
| `DATABASE_URL` | yes | Same Postgres as the rest of the repo |

## Run

```bash
cd apps/bizsafe-worker
go run ./cmd/import
```
