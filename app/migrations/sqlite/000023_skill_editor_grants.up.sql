-- SQLite mirror of 000111_skill_editor_grants.

CREATE TABLE IF NOT EXISTS skill_editor_grants (
    id           TEXT PRIMARY KEY,
    tenant_id    INTEGER NOT NULL,
    skill_id     TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_id   TEXT NOT NULL,
    created_by   TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_editor_grants_unique
    ON skill_editor_grants (skill_id, subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_skill_editor_grants_tenant
    ON skill_editor_grants (tenant_id, skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_editor_grants_subject
    ON skill_editor_grants (tenant_id, subject_type, subject_id);
