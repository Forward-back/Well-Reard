-- SQLite mirror of 000114_reading_runs.

ALTER TABLE tenants ADD COLUMN reading_model_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS reading_runs (
    id                 TEXT PRIMARY KEY,
    tenant_id          INTEGER NOT NULL,
    session_id         TEXT NOT NULL DEFAULT '',
    user_id            TEXT NOT NULL DEFAULT '',
    message_id         TEXT NOT NULL DEFAULT '',
    document_ids       TEXT NOT NULL DEFAULT '[]',
    skill_id           TEXT NOT NULL DEFAULT '',
    reading_model_id   TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'planning',
    plan               TEXT NOT NULL DEFAULT '{}',
    notes              TEXT NOT NULL DEFAULT '{}',
    final              TEXT NOT NULL DEFAULT '{}',
    current_section_id TEXT NOT NULL DEFAULT '',
    step_count         INTEGER NOT NULL DEFAULT 0,
    token_usage        INTEGER NOT NULL DEFAULT 0,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_runs_user
    ON reading_runs (tenant_id, user_id, session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_reading_runs_active
    ON reading_runs (tenant_id, user_id, session_id, status);

CREATE TABLE IF NOT EXISTS reading_run_steps (
    id            TEXT PRIMARY KEY,
    run_id        TEXT NOT NULL,
    seq           INTEGER NOT NULL,
    action        TEXT NOT NULL,
    section_id    TEXT NOT NULL DEFAULT '',
    question      TEXT NOT NULL DEFAULT '{}',
    user_feedback TEXT NOT NULL DEFAULT '',
    notes_patch   TEXT NOT NULL DEFAULT '{}',
    citations     TEXT NOT NULL DEFAULT '[]',
    status        TEXT NOT NULL DEFAULT 'done',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_run_steps_run
    ON reading_run_steps (run_id, seq);
