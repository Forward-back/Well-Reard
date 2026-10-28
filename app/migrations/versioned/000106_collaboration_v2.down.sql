-- Down migration 000106: collaboration v2.
UPDATE doc_change_requests SET status = 'open' WHERE status = 'pending_review';
UPDATE doc_change_requests SET status = 'accepted' WHERE status = 'merged';
UPDATE doc_change_requests SET status = 'rejected' WHERE status IN ('changes_requested', 'closed', 'draft');

DROP TABLE IF EXISTS chunk_citation_daily;
DROP TABLE IF EXISTS doc_anchor_comments;
DROP TABLE IF EXISTS doc_version_chunks;
DROP TABLE IF EXISTS doc_versions;
DROP TABLE IF EXISTS doc_cr_issues;
DROP TABLE IF EXISTS doc_cr_reviewers;

ALTER TABLE doc_cr_items DROP COLUMN IF EXISTS updated_at;
ALTER TABLE doc_cr_items DROP COLUMN IF EXISTS anchor_text;
ALTER TABLE doc_cr_items DROP COLUMN IF EXISTS viewed_by;
ALTER TABLE doc_cr_items DROP COLUMN IF EXISTS original_revision;

ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS draft_updated_at;
ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS tags;
ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS closed_at;
ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS closed_by;
ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS merged_at;
ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS merged_version_id;
ALTER TABLE doc_change_requests DROP COLUMN IF EXISTS base_version_id;

ALTER TABLE doc_issues DROP COLUMN IF EXISTS anchor_line_end;
ALTER TABLE doc_issues DROP COLUMN IF EXISTS anchor_line_start;
ALTER TABLE doc_issues DROP COLUMN IF EXISTS anchor_end;
ALTER TABLE doc_issues DROP COLUMN IF EXISTS anchor_start;
ALTER TABLE doc_issues DROP COLUMN IF EXISTS anchor_text;
ALTER TABLE doc_issues DROP COLUMN IF EXISTS issue_type;
