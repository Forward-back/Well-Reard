-- Migration: lite_embeddings BYTEA column (native vector store retirement, §2.15)
-- SQLite Lite stores vectors as float32 little-endian bytes in the embedding BLOB.
-- The table is also auto-migrated by the runtime retriever; create it if this
-- migration runs against a fresh database before that runtime path exists.
CREATE TABLE IF NOT EXISTS lite_embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    source_id TEXT NOT NULL,
    source_type INTEGER NOT NULL,
    chunk_id TEXT,
    knowledge_id TEXT,
    knowledge_base_id TEXT,
    tag_id TEXT,
    content TEXT NOT NULL,
    dimension INTEGER NOT NULL,
    embedding BLOB,
    is_enabled BOOLEAN DEFAULT TRUE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sqlite_emb_source
    ON lite_embeddings (source_id, source_type);
CREATE INDEX IF NOT EXISTS idx_lite_embeddings_chunk_id ON lite_embeddings (chunk_id);
CREATE INDEX IF NOT EXISTS idx_lite_embeddings_knowledge_id ON lite_embeddings (knowledge_id);
CREATE INDEX IF NOT EXISTS idx_lite_embeddings_knowledge_base_id ON lite_embeddings (knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_lite_embeddings_tag_id ON lite_embeddings (tag_id);
CREATE INDEX IF NOT EXISTS idx_lite_embeddings_is_enabled ON lite_embeddings (is_enabled);
