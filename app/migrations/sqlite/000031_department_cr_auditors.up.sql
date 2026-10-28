CREATE TABLE IF NOT EXISTS department_cr_auditors (
    tenant_id INTEGER NOT NULL,
    department_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    granted_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, department_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_department_cr_auditors_user
    ON department_cr_auditors (tenant_id, user_id);
