-- SQLite mirror of 000106_collaboration_v2.

ALTER TABLE doc_issues ADD COLUMN issue_type TEXT NOT NULL DEFAULT 'other';
ALTER TABLE doc_issues ADD COLUMN anchor_text TEXT NOT NULL DEFAULT '';
ALTER TABLE doc_issues ADD COLUMN anchor_start INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_issues ADD COLUMN anchor_end INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_issues ADD COLUMN anchor_line_start INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_issues ADD COLUMN anchor_line_end INTEGER NOT NULL DEFAULT 0;

UPDATE doc_change_requests SET status = 'pending_review' WHERE status = 'open';
UPDATE doc_change_requests SET status = 'merged' WHERE status = 'accepted';
UPDATE doc_change_requests SET status = 'changes_requested' WHERE status = 'rejected';

ALTER TABLE doc_change_requests ADD COLUMN base_version_id TEXT;
ALTER TABLE doc_change_requests ADD COLUMN merged_version_id TEXT;
ALTER TABLE doc_change_requests ADD COLUMN merged_at DATETIME;
ALTER TABLE doc_change_requests ADD COLUMN closed_by TEXT;
ALTER TABLE doc_change_requests ADD COLUMN closed_at DATETIME;
ALTER TABLE doc_change_requests ADD COLUMN tags TEXT NOT NULL DEFAULT '[]';
ALTER TABLE doc_change_requests ADD COLUMN draft_updated_at DATETIME;

ALTER TABLE doc_cr_items ADD COLUMN original_revision INTEGER NOT NULL DEFAULT 0;
ALTER TABLE doc_cr_items ADD COLUMN viewed_by TEXT NOT NULL DEFAULT '[]';
ALTER TABLE doc_cr_items ADD COLUMN anchor_text TEXT NOT NULL DEFAULT '';
ALTER TABLE doc_cr_items ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_doc_crs_status ON doc_change_requests(tenant_id, knowledge_base_id, status);
CREATE INDEX IF NOT EXISTS idx_doc_crs_knowledge_status ON doc_change_requests(knowledge_id, status);

CREATE TABLE IF NOT EXISTS doc_cr_reviewers (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    cr_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    comment TEXT NOT NULL DEFAULT '',
    decided_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_cr_reviewers_unique ON doc_cr_reviewers(cr_id, user_id);
CREATE INDEX IF NOT EXISTS idx_doc_cr_reviewers_cr ON doc_cr_reviewers(cr_id, status);
CREATE INDEX IF NOT EXISTS idx_doc_cr_reviewers_user ON doc_cr_reviewers(user_id, status);

CREATE TABLE IF NOT EXISTS doc_cr_issues (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    cr_id TEXT NOT NULL,
    issue_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_cr_issues_unique ON doc_cr_issues(cr_id, issue_id);
CREATE INDEX IF NOT EXISTS idx_doc_cr_issues_issue ON doc_cr_issues(issue_id);

CREATE TABLE IF NOT EXISTS doc_versions (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    knowledge_base_id TEXT NOT NULL,
    knowledge_id TEXT NOT NULL,
    version_seq INTEGER NOT NULL,
    version_label TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    chunk_count INTEGER NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_versions_knowledge_seq ON doc_versions(knowledge_id, version_seq);
CREATE INDEX IF NOT EXISTS idx_doc_versions_kb ON doc_versions(tenant_id, knowledge_base_id, knowledge_id);

CREATE TABLE IF NOT EXISTS doc_version_chunks (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    version_id TEXT NOT NULL,
    chunk_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    is_enabled INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_doc_version_chunks_version ON doc_version_chunks(version_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_doc_version_chunks_chunk ON doc_version_chunks(chunk_id);

CREATE TABLE IF NOT EXISTS doc_anchor_comments (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    knowledge_base_id TEXT NOT NULL,
    knowledge_id TEXT NOT NULL,
    chunk_id TEXT,
    parent_id TEXT,
    anchor_text TEXT NOT NULL DEFAULT '',
    anchor_start INTEGER NOT NULL DEFAULT 0,
    anchor_end INTEGER NOT NULL DEFAULT 0,
    anchor_line_start INTEGER NOT NULL DEFAULT 0,
    anchor_line_end INTEGER NOT NULL DEFAULT 0,
    body TEXT NOT NULL,
    resolved_at DATETIME,
    resolved_by TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_by TEXT,
    deleted_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_doc_anchor_comments_doc ON doc_anchor_comments(tenant_id, knowledge_id, created_at);
CREATE INDEX IF NOT EXISTS idx_doc_anchor_comments_chunk ON doc_anchor_comments(chunk_id);
CREATE INDEX IF NOT EXISTS idx_doc_anchor_comments_parent ON doc_anchor_comments(parent_id);

CREATE TABLE IF NOT EXISTS chunk_citation_daily (
    chunk_id TEXT NOT NULL,
    tenant_id INTEGER NOT NULL,
    stat_date TEXT NOT NULL,
    citation_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chunk_id, stat_date)
);
CREATE INDEX IF NOT EXISTS idx_chunk_citation_daily_tenant ON chunk_citation_daily(tenant_id, stat_date);
