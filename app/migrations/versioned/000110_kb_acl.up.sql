-- 000110: in-KB ACL rules + tenant ACL epoch.

CREATE TABLE IF NOT EXISTS kb_acl_rules (
    id             VARCHAR(36) PRIMARY KEY,
    tenant_id      BIGINT NOT NULL,
    kb_id          VARCHAR(36) NOT NULL,
    resource_type  VARCHAR(16) NOT NULL,
    resource_id    VARCHAR(64) NOT NULL,
    effect         VARCHAR(8) NOT NULL,
    subject_type   VARCHAR(16) NOT NULL,
    subject_id     VARCHAR(64) NOT NULL DEFAULT '',
    created_by     VARCHAR(36) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_acl_rules_unique
    ON kb_acl_rules (kb_id, resource_type, resource_id, effect, subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_kb_acl_rules_kb
    ON kb_acl_rules (tenant_id, kb_id);
CREATE INDEX IF NOT EXISTS idx_kb_acl_rules_subject
    ON kb_acl_rules (tenant_id, subject_type, subject_id);

CREATE TABLE IF NOT EXISTS tenant_acl_epochs (
    tenant_id  BIGINT PRIMARY KEY,
    epoch      BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO tenant_acl_epochs (tenant_id, epoch)
SELECT id, 0 FROM tenants
ON CONFLICT DO NOTHING;
