-- SQLite mirror of 000107_departments.

CREATE TABLE IF NOT EXISTS departments (
    id          TEXT PRIMARY KEY,
    tenant_id   INTEGER NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_default  INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_tenant_name
    ON departments (tenant_id, LOWER(name)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_tenant_default
    ON departments (tenant_id) WHERE is_default = 1 AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_departments_tenant
    ON departments (tenant_id) WHERE deleted_at IS NULL;

INSERT OR IGNORE INTO departments (id, tenant_id, name, description, is_default)
SELECT 'dept-default-' || t.id, t.id, '默认部门', '迁移自动创建的默认部门', 1
FROM tenants t
WHERE t.deleted_at IS NULL;

ALTER TABLE tenant_members ADD COLUMN department_id TEXT;
ALTER TABLE tenant_members ADD COLUMN department_role TEXT;

UPDATE tenant_members
SET department_id = 'dept-default-' || tenant_id
WHERE department_id IS NULL;

UPDATE tenant_members
SET department_role = CASE
    WHEN role IN ('owner', 'admin') THEN 'manager'
    WHEN role = 'contributor' AND EXISTS (
        SELECT 1 FROM knowledge_bases kb
        WHERE kb.creator_id = tenant_members.user_id
          AND kb.tenant_id = tenant_members.tenant_id
          AND kb.deleted_at IS NULL
    ) THEN 'knowledge_admin'
    WHEN role = 'contributor' THEN 'editor'
    ELSE 'viewer'
END
WHERE department_role IS NULL OR department_role = '';

UPDATE tenant_members
SET role = CASE WHEN role IN ('owner', 'admin') THEN 'company_admin' ELSE 'member' END
WHERE role IN ('owner', 'admin', 'contributor', 'viewer');

CREATE INDEX IF NOT EXISTS idx_tenant_members_department
    ON tenant_members (tenant_id, department_id);

ALTER TABLE knowledge_bases ADD COLUMN department_id TEXT;
UPDATE knowledge_bases
SET department_id = 'dept-default-' || tenant_id
WHERE department_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_department
    ON knowledge_bases (tenant_id, department_id);

CREATE TABLE IF NOT EXISTS kb_department_shares (
    id            TEXT PRIMARY KEY,
    tenant_id     INTEGER NOT NULL,
    kb_id         TEXT NOT NULL,
    department_id TEXT NOT NULL,
    created_by    TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_department_shares_unique
    ON kb_department_shares (kb_id, department_id);
CREATE INDEX IF NOT EXISTS idx_kb_department_shares_dept
    ON kb_department_shares (tenant_id, department_id);
CREATE INDEX IF NOT EXISTS idx_kb_department_shares_kb
    ON kb_department_shares (kb_id);

INSERT OR IGNORE INTO kb_department_shares (id, tenant_id, kb_id, department_id)
SELECT lower(hex(randomblob(16))), kb.tenant_id, kb.id, kb.department_id
FROM knowledge_bases kb
WHERE kb.department_id IS NOT NULL;

-- visibility is added by the runtime AutoMigrate; skip the legacy public-KB backfill here.
