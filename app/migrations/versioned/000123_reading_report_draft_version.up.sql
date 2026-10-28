-- 000123: optimistic versioning for edited reading report drafts.

ALTER TABLE reading_report_drafts
    ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
