-- Rollback: 000075_gaze_page_revisions
DO $$ BEGIN RAISE NOTICE '[Migration 000075] Reverting gaze page revisions schema'; END $$;

DROP TABLE IF EXISTS gaze_page_revisions;

ALTER TABLE gaze_pages DROP COLUMN IF EXISTS last_edit_source;
ALTER TABLE gaze_pages DROP COLUMN IF EXISTS last_editor_id;
