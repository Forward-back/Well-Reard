-- 000112: retrieval audit trail.
-- Best-effort per-request record of the ACL scope the engine applied.
-- Failure to write must never block retrieval.

CREATE TABLE IF NOT EXISTS retrieval_audit_logs (
    id               BIGSERIAL PRIMARY KEY,
    tenant_id        BIGINT NOT NULL,
    user_id          VARCHAR(36) NOT NULL DEFAULT '',
    principal_kind   VARCHAR(32) NOT NULL DEFAULT 'member',
    session_id       VARCHAR(36) NOT NULL DEFAULT '',
    query_digest     VARCHAR(128) NOT NULL DEFAULT '',
    kb_ids           TEXT NOT NULL DEFAULT '',
    scope_mode       VARCHAR(8) NOT NULL DEFAULT '',
    scope_version    BIGINT NOT NULL DEFAULT 0,
    scope_doc_count  INTEGER NOT NULL DEFAULT 0,
    hit_doc_ids      TEXT NOT NULL DEFAULT '',
    decision         VARCHAR(16) NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_retrieval_audit_tenant_time
    ON retrieval_audit_logs (tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_retrieval_audit_user_time
    ON retrieval_audit_logs (tenant_id, user_id, created_at);
