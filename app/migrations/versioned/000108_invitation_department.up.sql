-- 000108: carry department assignment on invitations.

ALTER TABLE tenant_invitations ADD COLUMN IF NOT EXISTS department_id VARCHAR(36);
ALTER TABLE tenant_invitations ADD COLUMN IF NOT EXISTS department_role VARCHAR(20);

CREATE INDEX IF NOT EXISTS idx_tenant_invitations_department
    ON tenant_invitations (tenant_id, department_id);
