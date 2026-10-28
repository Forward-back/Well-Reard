-- Migration: 000098_enterprise_metadata
-- Description: 企业级元数据(§2.14):knowledges 加生命周期/密级/部门 6 列,
-- chunks 反范式同 5 列(检索过滤/降权免 JOIN,superseded_by 不进 chunks),
-- 新增自定义字段定义表(维度 D 受控 KV)。
DO $$ BEGIN RAISE NOTICE '[Migration 000098] Adding enterprise metadata columns'; END $$;

ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(20) DEFAULT 'effective';
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS effective_date VARCHAR(50);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS expiry_date VARCHAR(50);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS superseded_by VARCHAR(100);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS sensitivity VARCHAR(20) DEFAULT 'internal';
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS department VARCHAR(100);

ALTER TABLE chunks ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(20) DEFAULT 'effective';
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS effective_date VARCHAR(50);
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS expiry_date VARCHAR(50);
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS sensitivity VARCHAR(20) DEFAULT 'internal';
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS department VARCHAR(100);

COMMENT ON COLUMN knowledges.lifecycle_status IS '生命周期: draft/effective/expired';
COMMENT ON COLUMN knowledges.superseded_by IS '替代本文号(替代链)';
COMMENT ON COLUMN knowledges.sensitivity IS '密级: internal/confidential';
COMMENT ON COLUMN knowledges.department IS '归属部门(静态枚举)';
COMMENT ON COLUMN chunks.lifecycle_status IS '生命周期反范式(检索降权免 JOIN)';
COMMENT ON COLUMN chunks.sensitivity IS '密级反范式(检索过滤免 JOIN)';

-- 维度 D:自定义字段定义表(受控 KV,人工配置字段名/类型/可选值)
CREATE TABLE IF NOT EXISTS knowledge_custom_field_defs (
    id          VARCHAR(36) PRIMARY KEY,
    tenant_id   BIGINT NOT NULL DEFAULT 0,
    name        VARCHAR(100) NOT NULL,
    field_type  VARCHAR(20) NOT NULL DEFAULT 'string',
    options     JSON,
    description VARCHAR(500),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

DO $$ BEGIN RAISE NOTICE '[Migration 000098] Enterprise metadata columns added successfully'; END $$;
