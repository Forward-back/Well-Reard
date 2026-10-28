-- Migration: gaze page outdated markers (wiki 版本过期同步, §2.15 对齐)
-- Description: gaze_pages gains outdated + superseded_by_slug. Summary pages
-- sharing one doc_number (UMS 文号) form a version chain: the newest page
-- stays current, older pages are flagged outdated and point at the page that
-- supersedes them via superseded_by_slug.
ALTER TABLE gaze_pages ADD COLUMN IF NOT EXISTS outdated BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE gaze_pages ADD COLUMN IF NOT EXISTS superseded_by_slug VARCHAR(255) NOT NULL DEFAULT '';
