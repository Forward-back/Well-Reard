DROP INDEX IF EXISTS idx_tenant_invitations_department;

ALTER TABLE tenant_invitations DROP COLUMN IF EXISTS department_role;
ALTER TABLE tenant_invitations DROP COLUMN IF EXISTS department_id;
