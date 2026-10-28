DROP INDEX IF EXISTS idx_tenant_skill_catalog_department;
DROP INDEX IF EXISTS uq_tenant_skill_catalog_dept_name;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_skill_catalog_name
    ON tenant_skill_catalog (tenant_id, name) WHERE deleted_at IS NULL;
ALTER TABLE tenant_skill_catalog DROP COLUMN department_id;
