-- SQLite down migration for 000018_collaboration_v2.
UPDATE doc_change_requests SET status = 'open' WHERE status = 'pending_review';
UPDATE doc_change_requests SET status = 'accepted' WHERE status = 'merged';
UPDATE doc_change_requests SET status = 'rejected' WHERE status IN ('changes_requested', 'closed', 'draft');

DROP TABLE IF EXISTS chunk_citation_daily;
DROP TABLE IF EXISTS doc_anchor_comments;
DROP TABLE IF EXISTS doc_version_chunks;
DROP TABLE IF EXISTS doc_versions;
DROP TABLE IF EXISTS doc_cr_issues;
DROP TABLE IF EXISTS doc_cr_reviewers;
