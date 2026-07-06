-- One-off backfill: copy gebiz_tender rows into the unified tender table.
-- Preserves original first_seen_at and any already-scraped details.
-- Idempotent (ON CONFLICT DO NOTHING); safe to re-run. Run before starting the
-- worker against `tender` so copied rows land first.
--
--   psql "$DATABASE_URL" -f apps/tender-worker/migrate_gebiz_to_tender.sql
--
-- If your gebiz_tender predates the detail-scrape feature and lacks the
-- details/details_fetched_at columns, replace those two SELECT items with NULL, NULL.

INSERT INTO tender (
  id, source, source_id, reference_no, title, link, category, agency, tender_type,
  published_date, closing_date, status, description, details, details_fetched_at,
  first_seen_at, last_seen_at
)
SELECT
  'gebiz:' || opportunity_id, 'gebiz', opportunity_id, reference_no, title, link,
  category, calling_entity, tender_type, published_date, closing_date, status,
  description, details, details_fetched_at, first_seen_at, last_seen_at
FROM gebiz_tender
ON CONFLICT (id) DO NOTHING;
