-- Migration: 000040_gaze_log_entries (rollback)
-- Description: Drop the gaze_log_entries table. No data migration was
--              performed on the way up, so rollback is a plain DROP.

DO $$ BEGIN RAISE NOTICE '[Migration 000040 DOWN] Reverting gaze_log_entries schema'; END $$;

DROP TABLE IF EXISTS gaze_log_entries;

DO $$ BEGIN RAISE NOTICE '[Migration 000040 DOWN] gaze_log_entries schema reverted successfully'; END $$;
