-- SQLite mirror of 000115_message_reading_run.

ALTER TABLE messages ADD COLUMN reading_run_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_messages_reading_run
    ON messages (reading_run_id);
