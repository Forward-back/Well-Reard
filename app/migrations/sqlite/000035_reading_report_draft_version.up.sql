-- SQLite mirror of 000123_reading_report_draft_version.

ALTER TABLE reading_report_drafts
    ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
