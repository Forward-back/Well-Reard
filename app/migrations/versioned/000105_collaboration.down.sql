-- Migration 000105 Down: drop collaboration tables and document/tenant sync columns.

DROP TABLE IF EXISTS doc_comments;
DROP TABLE IF EXISTS doc_cr_reviews;
DROP TABLE IF EXISTS doc_cr_items;
DROP TABLE IF EXISTS doc_change_requests;
DROP TABLE IF EXISTS doc_issues;
DROP SEQUENCE IF EXISTS doc_change_requests_seq_id_seq;
DROP SEQUENCE IF EXISTS doc_issues_seq_id_seq;

DROP INDEX IF EXISTS idx_knowledges_engine_sync;
DROP INDEX IF EXISTS idx_knowledges_created_by;
ALTER TABLE knowledges DROP COLUMN IF EXISTS engine_sync_error;
ALTER TABLE knowledges DROP COLUMN IF EXISTS engine_sync_completed_at;
ALTER TABLE knowledges DROP COLUMN IF EXISTS engine_sync_requested_at;
ALTER TABLE knowledges DROP COLUMN IF EXISTS engine_sync_status;
ALTER TABLE knowledges DROP COLUMN IF EXISTS created_by;

ALTER TABLE tenants DROP COLUMN IF EXISTS engine_sync_config;
