-- Migration 000106: collaboration v2 — five-state CR, drafts, reviewers,
-- document versions/rollbacks, issue anchors/types, selection comments.

-- ---------------------------------------------------------------------------
-- 1. Issue anchors + type
-- ---------------------------------------------------------------------------
ALTER TABLE doc_issues ADD COLUMN IF NOT EXISTS issue_type VARCHAR(20) NOT NULL DEFAULT 'other';
ALTER TABLE doc_issues ADD COLUMN IF NOT EXISTS anchor_text TEXT NOT NULL DEFAULT '';
ALTER TABLE doc_issues ADD COLUMN IF NOT EXISTS anchor_start INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_issues ADD COLUMN IF NOT EXISTS anchor_end INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_issues ADD COLUMN IF NOT EXISTS anchor_line_start INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_issues ADD COLUMN IF NOT EXISTS anchor_line_end INTEGER NOT NULL DEFAULT 0;

-- ---------------------------------------------------------------------------
-- 2. CR state machine + metadata
-- ---------------------------------------------------------------------------
UPDATE doc_change_requests SET status = 'pending_review' WHERE status = 'open';
UPDATE doc_change_requests SET status = 'merged' WHERE status = 'accepted';
UPDATE doc_change_requests SET status = 'changes_requested' WHERE status = 'rejected';
ALTER TABLE doc_change_requests ALTER COLUMN status SET DEFAULT 'pending_review';

ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS base_version_id VARCHAR(36);
ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS merged_version_id VARCHAR(36);
ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS merged_at TIMESTAMPTZ;
ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS closed_by VARCHAR(36);
ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;
ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE doc_change_requests ADD COLUMN IF NOT EXISTS draft_updated_at TIMESTAMPTZ;

ALTER TABLE doc_cr_items ADD COLUMN IF NOT EXISTS original_revision INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_cr_items ADD COLUMN IF NOT EXISTS viewed_by JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE doc_cr_items ADD COLUMN IF NOT EXISTS anchor_text TEXT NOT NULL DEFAULT '';
ALTER TABLE doc_cr_items ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_doc_crs_status ON doc_change_requests(tenant_id, knowledge_base_id, status);
CREATE INDEX IF NOT EXISTS idx_doc_crs_knowledge_status ON doc_change_requests(knowledge_id, status);

-- ---------------------------------------------------------------------------
-- 3. Reviewers (dynamic assignment, any approval is enough to merge)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS doc_cr_reviewers (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    cr_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    comment TEXT NOT NULL DEFAULT '',
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_cr_reviewers_unique ON doc_cr_reviewers(cr_id, user_id);
CREATE INDEX IF NOT EXISTS idx_doc_cr_reviewers_cr ON doc_cr_reviewers(cr_id, status);
CREATE INDEX IF NOT EXISTS idx_doc_cr_reviewers_user ON doc_cr_reviewers(user_id, status);

-- ---------------------------------------------------------------------------
-- 4. Explicit CR ↔ issue links (legacy linked-by-chunk remains as fallback)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS doc_cr_issues (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    cr_id VARCHAR(36) NOT NULL,
    issue_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_cr_issues_unique ON doc_cr_issues(cr_id, issue_id);
CREATE INDEX IF NOT EXISTS idx_doc_cr_issues_issue ON doc_cr_issues(issue_id);

-- ---------------------------------------------------------------------------
-- 5. Document versions + full chunk snapshots
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS doc_versions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    knowledge_id VARCHAR(36) NOT NULL,
    version_seq INTEGER NOT NULL,
    version_label VARCHAR(16) NOT NULL,
    source_type VARCHAR(24) NOT NULL,
    source_id VARCHAR(36) NOT NULL DEFAULT '',
    title VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    chunk_count INTEGER NOT NULL DEFAULT 0,
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_versions_knowledge_seq ON doc_versions(knowledge_id, version_seq);
CREATE INDEX IF NOT EXISTS idx_doc_versions_kb ON doc_versions(tenant_id, knowledge_base_id, knowledge_id);

CREATE TABLE IF NOT EXISTS doc_version_chunks (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    version_id VARCHAR(36) NOT NULL,
    chunk_id VARCHAR(36) NOT NULL,
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_doc_version_chunks_version ON doc_version_chunks(version_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_doc_version_chunks_chunk ON doc_version_chunks(chunk_id);

-- ---------------------------------------------------------------------------
-- 6. Selection comments (document content tab right rail)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS doc_anchor_comments (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    knowledge_id VARCHAR(36) NOT NULL,
    chunk_id VARCHAR(36),
    parent_id VARCHAR(36),
    anchor_text TEXT NOT NULL DEFAULT '',
    anchor_start INTEGER NOT NULL DEFAULT 0,
    anchor_end INTEGER NOT NULL DEFAULT 0,
    anchor_line_start INTEGER NOT NULL DEFAULT 0,
    anchor_line_end INTEGER NOT NULL DEFAULT 0,
    body TEXT NOT NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(36),
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by VARCHAR(36),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_doc_anchor_comments_doc ON doc_anchor_comments(tenant_id, knowledge_id, created_at);
CREATE INDEX IF NOT EXISTS idx_doc_anchor_comments_chunk ON doc_anchor_comments(chunk_id);
CREATE INDEX IF NOT EXISTS idx_doc_anchor_comments_parent ON doc_anchor_comments(parent_id);

-- ---------------------------------------------------------------------------
-- 7. Citation counters (app chunk ids after the ingest ownership mapping)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS chunk_citation_daily (
    chunk_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    stat_date DATE NOT NULL,
    citation_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chunk_id, stat_date)
);
CREATE INDEX IF NOT EXISTS idx_chunk_citation_daily_tenant ON chunk_citation_daily(tenant_id, stat_date);
