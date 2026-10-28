-- Migration: 000096_kb_visibility (down)
DO $$ BEGIN RAISE NOTICE '[Migration 000096] Dropping visibility column from knowledge_bases'; END $$;

ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS visibility;
