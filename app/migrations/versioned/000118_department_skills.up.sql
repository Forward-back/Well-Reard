-- 000118: scope skill definitions to one department.
ALTER TABLE tenant_skill_catalog
    ADD COLUMN IF NOT EXISTS department_id VARCHAR(36) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS uq_tenant_skill_catalog_name;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_skill_catalog_dept_name
    ON tenant_skill_catalog (tenant_id, department_id, name) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_skill_catalog_department
    ON tenant_skill_catalog (tenant_id, department_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN tenant_skill_catalog.department_id IS
    'Owning department; empty means legacy tenant-wide skill.';
