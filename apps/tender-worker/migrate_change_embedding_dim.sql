-- Widens tender.embedding from vector(128) (hashing-trick) to vector(768)
-- (Ollama nomic-embed-text). New installs get vector(768) directly from the
-- CREATE TABLE in store.go; this is for existing DBs.
--
-- pgvector cannot ALTER a vector column's dimension while an index depends on
-- it (the HNSW index was built for 128 dims), so the index must be dropped
-- first and recreated after. Existing embeddings are 128-dim hashing-trick
-- vectors that are NOT compatible with 768-dim Ollama vectors (different
-- model, different vector space) - ALTER COLUMN TYPE cannot losslessly
-- convert between dimensions, so this migration NULLs out every existing
-- embedding. Run the tender-worker --backfill-embeddings flag immediately
-- after applying this migration to re-populate them via Ollama.
DROP INDEX IF EXISTS tender_embedding_idx;

ALTER TABLE tender ALTER COLUMN embedding TYPE vector(768) USING NULL;

CREATE INDEX IF NOT EXISTS tender_embedding_idx ON tender USING hnsw (embedding vector_cosine_ops);
