-- Adds the AI delivery-timeline columns to an existing tender table.
-- New installs get these from the CREATE TABLE in store.go; this is for existing DBs.
ALTER TABLE tender ADD COLUMN IF NOT EXISTS ai_delivery_timeline    jsonb;
ALTER TABLE tender ADD COLUMN IF NOT EXISTS ai_delivery_timeline_at timestamptz;
