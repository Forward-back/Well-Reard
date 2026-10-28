-- Migration: 000061_gaze_page_hierarchy (rollback)

DROP INDEX IF EXISTS idx_gaze_pages_tree;
DROP INDEX IF EXISTS idx_gaze_pages_parent_slug;

ALTER TABLE gaze_pages DROP COLUMN IF EXISTS sort_order;
ALTER TABLE gaze_pages DROP COLUMN IF EXISTS depth;
ALTER TABLE gaze_pages DROP COLUMN IF EXISTS gaze_path;
ALTER TABLE gaze_pages DROP COLUMN IF EXISTS category_path;
ALTER TABLE gaze_pages DROP COLUMN IF EXISTS parent_slug;

ALTER TABLE gaze_pages DROP COLUMN IF EXISTS folder_id;

DROP TABLE IF EXISTS gaze_folders;

DROP INDEX IF EXISTS idx_gaze_pages_folder_id;