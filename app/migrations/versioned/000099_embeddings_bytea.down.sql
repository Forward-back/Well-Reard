-- Rollback of 000099: BYTEA -> halfvec.
-- Vector data is NOT recoverable (up.sql deletes all rows); this restores
-- the column type only, for schema symmetry. Requires the pgvector
-- extension to be present.
ALTER TABLE embeddings DROP COLUMN embedding;
ALTER TABLE embeddings ADD COLUMN embedding halfvec;
