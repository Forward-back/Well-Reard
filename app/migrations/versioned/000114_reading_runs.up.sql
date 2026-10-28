-- 000114: interactive document reading runs.

ALTER TABLE tenants ADD COLUMN reading_model_id VARCHAR(36) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS reading_runs (
    id               VARCHAR(36) PRIMARY KEY,
    tenant_id        BIGINT NOT NULL,
    session_id       VARCHAR(36) NOT NULL DEFAULT '',
    user_id          VARCHAR(36) NOT NULL DEFAULT '',
    message_id       VARCHAR(36) NOT NULL DEFAULT '',
    document_ids     JSONB NOT NULL DEFAULT '[]'::jsonb,
    skill_id         VARCHAR(36) NOT NULL DEFAULT '',
    reading_model_id VARCHAR(36) NOT NULL DEFAULT '',
    status           VARCHAR(16) NOT NULL DEFAULT 'planning',
    plan             JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes            JSONB NOT NULL DEFAULT '{}'::jsonb,
    final            JSONB NOT NULL DEFAULT '{}'::jsonb,
    current_section_id VARCHAR(64) NOT NULL DEFAULT '',
    step_count       INTEGER NOT NULL DEFAULT 0,
    token_usage      INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_runs_user
    ON reading_runs (tenant_id, user_id, session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_reading_runs_active
    ON reading_runs (tenant_id, user_id, session_id, status);

CREATE TABLE IF NOT EXISTS reading_run_steps (
    id            VARCHAR(36) PRIMARY KEY,
    run_id        VARCHAR(36) NOT NULL,
    seq           INTEGER NOT NULL,
    action        VARCHAR(24) NOT NULL,
    section_id    VARCHAR(64) NOT NULL DEFAULT '',
    question      JSONB NOT NULL DEFAULT '{}'::jsonb,
    user_feedback TEXT NOT NULL DEFAULT '',
    notes_patch   JSONB NOT NULL DEFAULT '{}'::jsonb,
    citations     JSONB NOT NULL DEFAULT '[]'::jsonb,
    status        VARCHAR(16) NOT NULL DEFAULT 'done',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_run_steps_run
    ON reading_run_steps (run_id, seq);
