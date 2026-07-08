-- Narrows tender.embedding from vector(384) (Ollama all-minilm) to
-- vector(256) (embed-service potion-base-8M — switched off Ollama onto the
-- self-contained internal embed-service). New installs get vector(256)
-- directly from the CREATE TABLE in store.go; this is for existing DBs.
--
-- pgvector cannot ALTER a vector column's dimension while an index depends on
-- it (the HNSW index was built for 384 dims), so the index must be dropped
-- first and recreated after. Existing 384-dim embeddings are NOT compatible
-- with 256-dim potion-base-8M vectors (different model, different vector
-- space), so this migration NULLs out every existing embedding. Run the
-- tender-worker --backfill-embeddings flag immediately after applying this
-- migration to re-populate them via embed-service.
DROP INDEX IF EXISTS tender_embedding_idx;

ALTER TABLE tender ALTER COLUMN embedding TYPE vector(256) USING NULL;

CREATE INDEX IF NOT EXISTS tender_embedding_idx ON tender USING hnsw (embedding vector_cosine_ops);
