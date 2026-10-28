-- Reverse of 000116_drop_user_resource_favorites.
CREATE TABLE IF NOT EXISTS user_resource_favorites (
    user_id        VARCHAR(36) NOT NULL,
    tenant_id      BIGINT      NOT NULL,
    resource_type  VARCHAR(16) NOT NULL,
    resource_id    VARCHAR(64) NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, tenant_id, resource_type, resource_id)
);

CREATE INDEX IF NOT EXISTS idx_user_resource_favorites_user_tenant_type_created_at
    ON user_resource_favorites (user_id, tenant_id, resource_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_resource_favorites_tenant_id
    ON user_resource_favorites (tenant_id);
