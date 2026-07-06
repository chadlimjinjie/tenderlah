-- Adds the AI bid-brief columns to an existing tender table.
-- New installs get these from the CREATE TABLE in store.go; this is for existing DBs.
ALTER TABLE tender ADD COLUMN IF NOT EXISTS ai_brief    jsonb;
ALTER TABLE tender ADD COLUMN IF NOT EXISTS ai_brief_at timestamptz;
