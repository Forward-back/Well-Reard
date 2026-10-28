-- SQLite mirror of 000121_reading_state_tables.

CREATE TABLE IF NOT EXISTS reading_questions (
    id                 TEXT PRIMARY KEY,
    run_id             TEXT NOT NULL,
    tenant_id          INTEGER NOT NULL,
    parent_id          TEXT NOT NULL DEFAULT '',
    section_id         TEXT NOT NULL DEFAULT '',
    round              INTEGER NOT NULL DEFAULT 0,
    text               TEXT NOT NULL DEFAULT '',
    recommended_answer TEXT NOT NULL DEFAULT '',
    why                TEXT NOT NULL DEFAULT '',
    search_query       TEXT NOT NULL DEFAULT '',
    evidence           TEXT NOT NULL DEFAULT '[]',
    answer             TEXT NOT NULL DEFAULT '',
    answer_source      TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'pending',
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_questions_run
    ON reading_questions (run_id, round, created_at);
CREATE INDEX IF NOT EXISTS idx_reading_questions_status
    ON reading_questions (run_id, status);

CREATE TABLE IF NOT EXISTS reading_conclusions (
    id          TEXT PRIMARY KEY,
    run_id      TEXT NOT NULL,
    tenant_id   INTEGER NOT NULL,
    section_id  TEXT NOT NULL DEFAULT '',
    round       INTEGER NOT NULL DEFAULT 0,
    text        TEXT NOT NULL DEFAULT '',
    evidence    TEXT NOT NULL DEFAULT '[]',
    status      TEXT NOT NULL DEFAULT 'candidate',
    source      TEXT NOT NULL DEFAULT 'model',
    gates       TEXT NOT NULL DEFAULT '{}',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_conclusions_run
    ON reading_conclusions (run_id, status, created_at);

CREATE TABLE IF NOT EXISTS reading_terms (
    id          TEXT PRIMARY KEY,
    run_id      TEXT NOT NULL,
    tenant_id   INTEGER NOT NULL,
    term        TEXT NOT NULL DEFAULT '',
    definition  TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT '[]',
    canonical   INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'candidate',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_terms_run
    ON reading_terms (run_id, created_at);

CREATE TABLE IF NOT EXISTS reading_contradictions (
    id          TEXT PRIMARY KEY,
    run_id      TEXT NOT NULL,
    tenant_id   INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    evidence_a  TEXT NOT NULL DEFAULT '[]',
    evidence_b  TEXT NOT NULL DEFAULT '[]',
    status      TEXT NOT NULL DEFAULT 'open',
    resolution  TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reading_contradictions_run
    ON reading_contradictions (run_id, status, created_at);
