-- SQLite mirror of 000113_skill_injection.

CREATE TABLE IF NOT EXISTS tenant_skill_catalog (
    id            TEXT PRIMARY KEY,
    tenant_id     INTEGER NOT NULL,
    name          TEXT NOT NULL,
    version       TEXT,
    description   TEXT,
    instructions  TEXT,
    bundle_ref    TEXT,
    bundle_sha256 TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_skill_catalog_name
    ON tenant_skill_catalog (tenant_id, name) WHERE deleted_at IS NULL;

ALTER TABLE tenant_skill_catalog ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE messages ADD COLUMN skill TEXT;
