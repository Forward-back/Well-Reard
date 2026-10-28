-- SQLite mirror of 000109_knowledge_folders.

CREATE TABLE IF NOT EXISTS knowledge_folders (
    id          TEXT PRIMARY KEY,
    tenant_id   INTEGER NOT NULL,
    kb_id       TEXT NOT NULL,
    parent_id   TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    path        TEXT NOT NULL DEFAULT '',
    depth       INTEGER NOT NULL DEFAULT 0,
    is_root     INTEGER NOT NULL DEFAULT 0,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_folders_kb_path
    ON knowledge_folders (kb_id, path) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_folders_sibling
    ON knowledge_folders (kb_id, parent_id, name)
    WHERE deleted_at IS NULL AND is_root = 0;
CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_folders_root
    ON knowledge_folders (kb_id) WHERE deleted_at IS NULL AND is_root = 1;
CREATE INDEX IF NOT EXISTS idx_knowledge_folders_kb
    ON knowledge_folders (kb_id) WHERE deleted_at IS NULL;

INSERT OR IGNORE INTO knowledge_folders (id, tenant_id, kb_id, parent_id, name, path, depth, is_root, created_by)
SELECT 'folder-root-' || kb.id, kb.tenant_id, kb.id, '', '', '', 0, 1, kb.creator_id
FROM knowledge_bases kb
WHERE kb.deleted_at IS NULL;

-- Materialize every prefix of every existing folder_path.
WITH RECURSIVE
paths(tenant_id, kb_id, folder_path) AS (
    SELECT DISTINCT tenant_id, knowledge_base_id, folder_path
    FROM knowledges
    WHERE deleted_at IS NULL AND folder_path <> ''
),
prefixes(tenant_id, kb_id, path, prefix, parent_prefix, rest, depth) AS (
    SELECT tenant_id, kb_id, folder_path, '', '', folder_path, 0
    FROM paths
    UNION ALL
    SELECT
        tenant_id,
        kb_id,
        path,
        CASE
            WHEN prefix = '' THEN
                CASE WHEN instr(rest, '/') > 0 THEN substr(rest, 1, instr(rest, '/') - 1) ELSE rest END
            ELSE
                prefix || '/' || CASE WHEN instr(rest, '/') > 0 THEN substr(rest, 1, instr(rest, '/') - 1) ELSE rest END
        END,
        prefix,
        CASE WHEN instr(rest, '/') > 0 THEN substr(rest, instr(rest, '/') + 1) ELSE '' END,
        depth + 1
    FROM prefixes
    WHERE rest <> ''
)
INSERT OR IGNORE INTO knowledge_folders (id, tenant_id, kb_id, parent_id, name, path, depth, is_root, created_by)
SELECT
    'fd-' || lower(hex(kb_id || ':' || prefix)),
    tenant_id,
    kb_id,
    CASE
        WHEN parent_prefix = '' THEN 'folder-root-' || kb_id
        ELSE 'fd-' || lower(hex(kb_id || ':' || parent_prefix))
    END,
    CASE
        WHEN parent_prefix = '' THEN prefix
        ELSE substr(prefix, length(parent_prefix) + 2)
    END,
    prefix,
    depth,
    0,
    ''
FROM prefixes
WHERE prefix <> '';

ALTER TABLE knowledges ADD COLUMN folder_id TEXT NOT NULL DEFAULT '';

UPDATE knowledges
SET folder_id = COALESCE(
    (
        SELECT f.id
        FROM knowledge_folders f
        WHERE f.kb_id = knowledges.knowledge_base_id
          AND f.path = knowledges.folder_path
          AND f.deleted_at IS NULL
        LIMIT 1
    ),
    'folder-root-' || knowledge_base_id
);

CREATE INDEX IF NOT EXISTS idx_knowledges_folder_id
    ON knowledges (tenant_id, knowledge_base_id, folder_id);
