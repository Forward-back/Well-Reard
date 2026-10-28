-- 000122: user-edited reading report drafts.

CREATE TABLE IF NOT EXISTS reading_report_drafts (
    run_id     VARCHAR(36) PRIMARY KEY,
    tenant_id  BIGINT NOT NULL,
    user_id    VARCHAR(36) NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_report_drafts_user
    ON reading_report_drafts (tenant_id, user_id);
