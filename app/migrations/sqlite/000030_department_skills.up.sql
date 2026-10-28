ALTER TABLE tenant_skill_catalog ADD COLUMN department_id TEXT NOT NULL DEFAULT '';
DROP INDEX IF EXISTS uq_tenant_skill_catalog_name;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_skill_catalog_dept_name
    ON tenant_skill_catalog (tenant_id, department_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_skill_catalog_department
    ON tenant_skill_catalog (tenant_id, department_id) WHERE deleted_at IS NULL;
