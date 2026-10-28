-- 000111: per-skill editor grants.
-- Grants extend edit authority to one user or department without changing
-- global roles. They never grant KB, ACL, sandbox, or MCP access.

CREATE TABLE IF NOT EXISTS skill_editor_grants (
    id           VARCHAR(36) PRIMARY KEY,
    tenant_id    BIGINT NOT NULL,
    skill_id     VARCHAR(36) NOT NULL,
    subject_type VARCHAR(16) NOT NULL,
    subject_id   VARCHAR(64) NOT NULL,
    created_by   VARCHAR(36) NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_editor_grants_unique
    ON skill_editor_grants (skill_id, subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_skill_editor_grants_tenant
    ON skill_editor_grants (tenant_id, skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_editor_grants_subject
    ON skill_editor_grants (tenant_id, subject_type, subject_id);
