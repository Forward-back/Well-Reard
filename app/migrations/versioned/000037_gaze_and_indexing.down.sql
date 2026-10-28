-- Migration: 000037_gaze_and_indexing (rollback)
-- Description: Reverse the gaze + indexing schema in the opposite order of the up migration.
DO $$ BEGIN RAISE NOTICE '[Migration 000037 DOWN] Reverting gaze + indexing schema'; END $$;

ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS indexing_strategy;

DROP TABLE IF EXISTS gaze_page_issues;

DROP TABLE IF EXISTS gaze_folders;

DROP TABLE IF EXISTS gaze_pages;

ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS gaze_config;

DO $$ BEGIN RAISE NOTICE '[Migration 000037 DOWN] gaze + indexing schema reverted successfully'; END $$;
