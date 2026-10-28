-- SQLite mirror of 000112_retrieval_audit.

CREATE TABLE IF NOT EXISTS retrieval_audit_logs (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id        INTEGER NOT NULL,
    user_id          TEXT NOT NULL DEFAULT '',
    principal_kind   TEXT NOT NULL DEFAULT 'member',
    session_id       TEXT NOT NULL DEFAULT '',
    query_digest     TEXT NOT NULL DEFAULT '',
    kb_ids           TEXT NOT NULL DEFAULT '',
    scope_mode       TEXT NOT NULL DEFAULT '',
    scope_version    INTEGER NOT NULL DEFAULT 0,
    scope_doc_count  INTEGER NOT NULL DEFAULT 0,
    hit_doc_ids      TEXT NOT NULL DEFAULT '',
    decision         TEXT NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_retrieval_audit_tenant_time
    ON retrieval_audit_logs (tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_retrieval_audit_user_time
    ON retrieval_audit_logs (tenant_id, user_id, created_at);
