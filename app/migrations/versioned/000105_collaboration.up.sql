-- Migration 000105: Git 式文档维护（需求一）
-- Description: 协作表（issue / CR / CR 条目 / 审批历史 / 评论）、文档上传者与
-- YLSW 待重推状态、租户级每日重推配置。批2 需求一。

-- ============================================================================
-- Section 1: 文档上传者 + 引擎同步状态
-- ============================================================================

ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS created_by VARCHAR(36);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS engine_sync_status VARCHAR(20) NOT NULL DEFAULT 'synced';
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS engine_sync_requested_at TIMESTAMPTZ;
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS engine_sync_completed_at TIMESTAMPTZ;
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS engine_sync_error TEXT NOT NULL DEFAULT '';

-- 存量文档没有独立上传者，按父 KB 创建者近似回填。
UPDATE knowledges k
SET created_by = kb.creator_id
FROM knowledge_bases kb
WHERE k.created_by IS NULL
  AND kb.id = k.knowledge_base_id
  AND kb.creator_id IS NOT NULL
  AND kb.creator_id <> '';

CREATE INDEX IF NOT EXISTS idx_knowledges_created_by ON knowledges(created_by);
CREATE INDEX IF NOT EXISTS idx_knowledges_engine_sync ON knowledges(tenant_id, knowledge_base_id, engine_sync_status);

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS engine_sync_config JSONB;

-- ============================================================================
-- Section 2: 文档 issue
-- ============================================================================

CREATE SEQUENCE IF NOT EXISTS doc_issues_seq_id_seq START WITH 1;
CREATE SEQUENCE IF NOT EXISTS doc_change_requests_seq_id_seq START WITH 1;

CREATE TABLE IF NOT EXISTS doc_issues (
    id VARCHAR(36) PRIMARY KEY,
    seq_id BIGINT NOT NULL DEFAULT nextval('doc_issues_seq_id_seq'),
    tenant_id BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    knowledge_id VARCHAR(36) NOT NULL,
    chunk_id VARCHAR(36),
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    anchor_orphaned BOOLEAN NOT NULL DEFAULT FALSE,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_by VARCHAR(36),
    closed_at TIMESTAMPTZ,
    deleted_by VARCHAR(36),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_issues_seq_id ON doc_issues(seq_id);
CREATE INDEX IF NOT EXISTS idx_doc_issues_kb_status ON doc_issues(tenant_id, knowledge_base_id, status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_doc_issues_knowledge ON doc_issues(knowledge_id);
CREATE INDEX IF NOT EXISTS idx_doc_issues_chunk ON doc_issues(chunk_id);
CREATE INDEX IF NOT EXISTS idx_doc_issues_creator ON doc_issues(created_by);

-- ============================================================================
-- Section 3: 变更请求（CR）
-- ============================================================================

CREATE TABLE IF NOT EXISTS doc_change_requests (
    id VARCHAR(36) PRIMARY KEY,
    seq_id BIGINT NOT NULL DEFAULT nextval('doc_change_requests_seq_id_seq'),
    tenant_id BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    knowledge_id VARCHAR(36) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    anchor_orphaned BOOLEAN NOT NULL DEFAULT FALSE,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_by VARCHAR(36),
    reviewed_at TIMESTAMPTZ,
    review_comment TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_crs_seq_id ON doc_change_requests(seq_id);
CREATE INDEX IF NOT EXISTS idx_doc_crs_kb_status ON doc_change_requests(tenant_id, knowledge_base_id, status);
CREATE INDEX IF NOT EXISTS idx_doc_crs_knowledge ON doc_change_requests(knowledge_id);
CREATE INDEX IF NOT EXISTS idx_doc_crs_creator ON doc_change_requests(created_by);

CREATE TABLE IF NOT EXISTS doc_cr_items (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    cr_id VARCHAR(36) NOT NULL,
    chunk_id VARCHAR(36) NOT NULL,
    original_content TEXT NOT NULL,
    suggested_content TEXT NOT NULL,
    seq INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_cr_items_cr ON doc_cr_items(cr_id, seq);
CREATE INDEX IF NOT EXISTS idx_doc_cr_items_chunk ON doc_cr_items(chunk_id);

CREATE TABLE IF NOT EXISTS doc_cr_reviews (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    cr_id VARCHAR(36) NOT NULL,
    action VARCHAR(20) NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    reviewed_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_cr_reviews_cr ON doc_cr_reviews(cr_id, created_at);

-- ============================================================================
-- Section 4: 评论（issue 与 CR 共用）
-- ============================================================================

CREATE TABLE IF NOT EXISTS doc_comments (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    target_type VARCHAR(16) NOT NULL,
    target_id VARCHAR(36) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by VARCHAR(36),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_doc_comments_target ON doc_comments(target_type, target_id, created_at);
CREATE INDEX IF NOT EXISTS idx_doc_comments_kb ON doc_comments(tenant_id, knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_doc_comments_creator ON doc_comments(created_by);

DO $$ BEGIN RAISE NOTICE '[Migration 000105] collaboration tables created'; END $$;
