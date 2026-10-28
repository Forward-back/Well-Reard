-- Migration: 000096_kb_visibility
-- Description: Add visibility column to knowledge_bases (private/internal/public, 消费者门户发布标记)
DO $$ BEGIN RAISE NOTICE '[Migration 000096] Adding visibility column to knowledge_bases'; END $$;

ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS visibility VARCHAR(16) NOT NULL DEFAULT 'internal';

COMMENT ON COLUMN knowledge_bases.visibility IS 'KB 对外可见性: private/internal/public(仅 Admin+ 可变更,见改造方案 §2.13)';
