-- Narrows tender.embedding from vector(768) (Ollama nomic-embed-text) to
-- vector(384) (Ollama all-minilm — switched to cut the embedding container's
-- compute footprint). New installs get vector(384) directly from the
-- CREATE TABLE in store.go; this is for existing DBs.
--
-- pgvector cannot ALTER a vector column's dimension while an index depends on
-- it (the HNSW index was built for 768 dims), so the index must be dropped
-- first and recreated after. Existing 768-dim embeddings are NOT compatible
-- with 384-dim all-minilm vectors (different model, different vector space),
-- so this migration NULLs out every existing embedding. Run the
-- tender-worker --backfill-embeddings flag immediately after applying this
-- migration to re-populate them via Ollama.
DROP INDEX IF EXISTS tender_embedding_idx;

ALTER TABLE tender ALTER COLUMN embedding TYPE vector(384) USING NULL;

CREATE INDEX IF NOT EXISTS tender_embedding_idx ON tender USING hnsw (embedding vector_cosine_ops);
