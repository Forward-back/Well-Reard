-- SQLite mirror of 000122_reading_report_drafts.

CREATE TABLE IF NOT EXISTS reading_report_drafts (
    run_id     TEXT PRIMARY KEY,
    tenant_id  INTEGER NOT NULL,
    user_id    TEXT NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_report_drafts_user
    ON reading_report_drafts (tenant_id, user_id);
