-- 000113: skill enable flag + per-assistant-message skill metadata.

ALTER TABLE tenant_skill_catalog ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE messages ADD COLUMN skill JSONB;
