-- 000115: link chat messages to interactive reading runs.

ALTER TABLE messages ADD COLUMN reading_run_id VARCHAR(36) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_messages_reading_run
    ON messages (reading_run_id);
