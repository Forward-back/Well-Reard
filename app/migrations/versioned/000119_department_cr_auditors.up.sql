CREATE TABLE IF NOT EXISTS department_cr_auditors (
    tenant_id BIGINT NOT NULL,
    department_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    granted_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, department_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_department_cr_auditors_user
    ON department_cr_auditors (tenant_id, user_id);
