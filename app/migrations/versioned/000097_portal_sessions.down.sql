-- Migration: 000097_portal_sessions (down)
DO $$ BEGIN RAISE NOTICE '[Migration 000097] Dropping portal_sessions / portal_messages'; END $$;

DROP TABLE IF EXISTS portal_messages;
DROP TABLE IF EXISTS portal_sessions;
