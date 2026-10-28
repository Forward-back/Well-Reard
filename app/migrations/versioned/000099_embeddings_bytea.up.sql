-- Migration: embeddings BYTEA (native vector store retirement, §2.15)
-- Description: embeddings.embedding halfvec -> BYTEA. The app no longer
-- performs vector search in-database; similarity is computed in Go over
-- BYTEA-encoded float32 vectors (see retriever/postgres VectorRetrieve).
-- Only FAQ chunks and ephemeral web-search KBs still write vectors;
-- regular knowledge chunks were migrated to the YLSW engine (§2.2).
--
-- Stored halfvec rows are dropped: they are pure dual-write redundancy for
-- regular chunks (YLSW holds the authoritative copy) and there were no FAQ
-- chunks persisted at migration time. pgvector's halfvec type has no cast
-- to or from bytea, so the column is dropped and re-added instead of
-- ALTERed in place.

DELETE FROM embeddings;

-- HNSW expression indexes must go before the column they reference.
DROP INDEX IF EXISTS embeddings_embedding_idx_3584;
DROP INDEX IF EXISTS embeddings_embedding_idx_798;

ALTER TABLE embeddings DROP COLUMN embedding;
ALTER TABLE embeddings ADD COLUMN embedding BYTEA;
