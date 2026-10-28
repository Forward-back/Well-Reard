-- Migration 000107: departments + company/department role split + KB ownership.

-- ---------------------------------------------------------------------------
-- 1. Departments
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS departments (
    id          VARCHAR(36) PRIMARY KEY,
    tenant_id   BIGINT NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_tenant_name
    ON departments (tenant_id, LOWER(name)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_tenant_default
    ON departments (tenant_id) WHERE is_default = TRUE AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_departments_tenant
    ON departments (tenant_id) WHERE deleted_at IS NULL;

-- One default department per tenant. Deterministic id keeps the migration
-- idempotent and lets backfills reference it without a lookup join.
INSERT INTO departments (id, tenant_id, name, description, is_default)
SELECT 'dept-default-' || t.id::text, t.id, '默认部门', '迁移自动创建的默认部门', TRUE
FROM tenants t
WHERE t.deleted_at IS NULL
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2. Membership department assignment + role collapse
-- ---------------------------------------------------------------------------
ALTER TABLE tenant_members ADD COLUMN IF NOT EXISTS department_id VARCHAR(36);
ALTER TABLE tenant_members ADD COLUMN IF NOT EXISTS department_role VARCHAR(20);

UPDATE tenant_members
SET department_id = 'dept-default-' || tenant_id::text
WHERE department_id IS NULL;

-- Derive department roles from the legacy tenant role before collapsing it.
UPDATE tenant_members tm
SET department_role = CASE
    WHEN tm.role IN ('owner', 'admin') THEN 'manager'
    WHEN tm.role = 'contributor' AND EXISTS (
        SELECT 1 FROM knowledge_bases kb
        WHERE kb.creator_id = tm.user_id
          AND kb.tenant_id = tm.tenant_id
          AND kb.deleted_at IS NULL
    ) THEN 'knowledge_admin'
    WHEN tm.role = 'contributor' THEN 'editor'
    ELSE 'viewer'
END
WHERE tm.department_role IS NULL OR tm.department_role = '';

UPDATE tenant_members
SET role = CASE WHEN role IN ('owner', 'admin') THEN 'company_admin' ELSE 'member' END
WHERE role IN ('owner', 'admin', 'contributor', 'viewer');

CREATE INDEX IF NOT EXISTS idx_tenant_members_department
    ON tenant_members (tenant_id, department_id);

-- ---------------------------------------------------------------------------
-- 3. Knowledge base owning department
-- ---------------------------------------------------------------------------
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS department_id VARCHAR(36);
UPDATE knowledge_bases
SET department_id = 'dept-default-' || tenant_id::text
WHERE department_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_department
    ON knowledge_bases (tenant_id, department_id);

-- ---------------------------------------------------------------------------
-- 4. Department share scope (materialized department set)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kb_department_shares (
    id            VARCHAR(36) PRIMARY KEY,
    tenant_id     BIGINT NOT NULL,
    kb_id         VARCHAR(36) NOT NULL,
    department_id VARCHAR(36) NOT NULL,
    created_by    VARCHAR(36) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_department_shares_unique
    ON kb_department_shares (kb_id, department_id);
CREATE INDEX IF NOT EXISTS idx_kb_department_shares_dept
    ON kb_department_shares (tenant_id, department_id);
CREATE INDEX IF NOT EXISTS idx_kb_department_shares_kb
    ON kb_department_shares (kb_id);

-- Every KB is visible to its owning department by default.
INSERT INTO kb_department_shares (id, tenant_id, kb_id, department_id)
SELECT md5(kb.id || ':' || kb.department_id), kb.tenant_id, kb.id, kb.department_id
FROM knowledge_bases kb
WHERE kb.department_id IS NOT NULL
ON CONFLICT (kb_id, department_id) DO NOTHING;

-- public (consumer portal) KBs are also company-wide for internal sharing.
INSERT INTO kb_department_shares (id, tenant_id, kb_id, department_id)
SELECT md5(kb.id || ':' || d.id), kb.tenant_id, kb.id, d.id
FROM knowledge_bases kb
JOIN departments d ON d.tenant_id = kb.tenant_id AND d.deleted_at IS NULL
WHERE kb.visibility = 'public'
ON CONFLICT (kb_id, department_id) DO NOTHING;
