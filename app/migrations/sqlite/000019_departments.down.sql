-- Rollback 000019. Role collapse is lossy: company_admin -> admin,
-- member -> viewer.

DROP TABLE IF EXISTS kb_department_shares;

DROP INDEX IF EXISTS idx_knowledge_bases_department;
ALTER TABLE knowledge_bases DROP COLUMN department_id;

DROP INDEX IF EXISTS idx_tenant_members_department;
UPDATE tenant_members
SET role = CASE WHEN role = 'company_admin' THEN 'admin' ELSE 'viewer' END;
ALTER TABLE tenant_members DROP COLUMN department_role;
ALTER TABLE tenant_members DROP COLUMN department_id;

DROP TABLE IF EXISTS departments;
