-- SQLite mirror of 000110_kb_acl.

CREATE TABLE IF NOT EXISTS kb_acl_rules (
    id             TEXT PRIMARY KEY,
    tenant_id      INTEGER NOT NULL,
    kb_id          TEXT NOT NULL,
    resource_type  TEXT NOT NULL,
    resource_id    TEXT NOT NULL,
    effect         TEXT NOT NULL,
    subject_type   TEXT NOT NULL,
    subject_id     TEXT NOT NULL DEFAULT '',
    created_by     TEXT NOT NULL DEFAULT '',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_acl_rules_unique
    ON kb_acl_rules (kb_id, resource_type, resource_id, effect, subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_kb_acl_rules_kb
    ON kb_acl_rules (tenant_id, kb_id);
CREATE INDEX IF NOT EXISTS idx_kb_acl_rules_subject
    ON kb_acl_rules (tenant_id, subject_type, subject_id);

CREATE TABLE IF NOT EXISTS tenant_acl_epochs (
    tenant_id  INTEGER PRIMARY KEY,
    epoch      INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO tenant_acl_epochs (tenant_id, epoch)
SELECT id, 0 FROM tenants;
