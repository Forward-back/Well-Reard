-- SQLite mirror of 000108_invitation_department.

ALTER TABLE tenant_invitations ADD COLUMN department_id TEXT;
ALTER TABLE tenant_invitations ADD COLUMN department_role TEXT;

CREATE INDEX IF NOT EXISTS idx_tenant_invitations_department
    ON tenant_invitations (tenant_id, department_id);
