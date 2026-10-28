-- 000121: structured state for interactive document reading.

CREATE TABLE IF NOT EXISTS reading_questions (
    id                 VARCHAR(36) PRIMARY KEY,
    run_id             VARCHAR(36) NOT NULL,
    tenant_id          BIGINT NOT NULL,
    parent_id          VARCHAR(36) NOT NULL DEFAULT '',
    section_id         VARCHAR(64) NOT NULL DEFAULT '',
    round              INTEGER NOT NULL DEFAULT 0,
    text               TEXT NOT NULL DEFAULT '',
    recommended_answer TEXT NOT NULL DEFAULT '',
    why                TEXT NOT NULL DEFAULT '',
    search_query       TEXT NOT NULL DEFAULT '',
    evidence           JSONB NOT NULL DEFAULT '[]'::jsonb,
    answer             TEXT NOT NULL DEFAULT '',
    answer_source      VARCHAR(16) NOT NULL DEFAULT '',
    status             VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_questions_run
    ON reading_questions (run_id, round, created_at);
CREATE INDEX IF NOT EXISTS idx_reading_questions_status
    ON reading_questions (run_id, status);

CREATE TABLE IF NOT EXISTS reading_conclusions (
    id          VARCHAR(36) PRIMARY KEY,
    run_id      VARCHAR(36) NOT NULL,
    tenant_id   BIGINT NOT NULL,
    section_id  VARCHAR(64) NOT NULL DEFAULT '',
    round       INTEGER NOT NULL DEFAULT 0,
    text        TEXT NOT NULL DEFAULT '',
    evidence    JSONB NOT NULL DEFAULT '[]'::jsonb,
    status      VARCHAR(20) NOT NULL DEFAULT 'candidate',
    source      VARCHAR(16) NOT NULL DEFAULT 'model',
    gates       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_conclusions_run
    ON reading_conclusions (run_id, status, created_at);

CREATE TABLE IF NOT EXISTS reading_terms (
    id          VARCHAR(36) PRIMARY KEY,
    run_id      VARCHAR(36) NOT NULL,
    tenant_id   BIGINT NOT NULL,
    term        VARCHAR(200) NOT NULL DEFAULT '',
    definition  TEXT NOT NULL DEFAULT '',
    source      JSONB NOT NULL DEFAULT '[]'::jsonb,
    canonical   BOOLEAN NOT NULL DEFAULT FALSE,
    status      VARCHAR(20) NOT NULL DEFAULT 'candidate',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_terms_run
    ON reading_terms (run_id, created_at);

CREATE TABLE IF NOT EXISTS reading_contradictions (
    id          VARCHAR(36) PRIMARY KEY,
    run_id      VARCHAR(36) NOT NULL,
    tenant_id   BIGINT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    evidence_a  JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence_b  JSONB NOT NULL DEFAULT '[]'::jsonb,
    status      VARCHAR(20) NOT NULL DEFAULT 'open',
    resolution  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_contradictions_run
    ON reading_contradictions (run_id, status, created_at);
