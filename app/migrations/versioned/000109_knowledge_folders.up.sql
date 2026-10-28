-- 000109: real knowledge folders + document folder_id backfill.
-- folder_path stays for one compatibility release; folder_id is authoritative.

CREATE TABLE IF NOT EXISTS knowledge_folders (
    id          VARCHAR(64) PRIMARY KEY,
    tenant_id   BIGINT NOT NULL,
    kb_id       VARCHAR(36) NOT NULL,
    parent_id   VARCHAR(64) NOT NULL DEFAULT '',
    name        VARCHAR(128) NOT NULL,
    path        VARCHAR(1024) NOT NULL DEFAULT '',
    depth       INTEGER NOT NULL DEFAULT 0,
    is_root     BOOLEAN NOT NULL DEFAULT FALSE,
    created_by  VARCHAR(36) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_folders_kb_path
    ON knowledge_folders (kb_id, path) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_folders_sibling
    ON knowledge_folders (kb_id, parent_id, name)
    WHERE deleted_at IS NULL AND is_root = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_folders_root
    ON knowledge_folders (kb_id) WHERE deleted_at IS NULL AND is_root = TRUE;
CREATE INDEX IF NOT EXISTS idx_knowledge_folders_kb
    ON knowledge_folders (kb_id) WHERE deleted_at IS NULL;

-- One root folder per knowledge base.
INSERT INTO knowledge_folders (id, tenant_id, kb_id, parent_id, name, path, depth, is_root, created_by)
SELECT 'folder-root-' || kb.id, kb.tenant_id, kb.id, '', '', '', 0, TRUE, kb.creator_id
FROM knowledge_bases kb
WHERE kb.deleted_at IS NULL
ON CONFLICT DO NOTHING;

-- Materialize every prefix of every existing folder_path.
WITH RECURSIVE paths AS (
    SELECT DISTINCT tenant_id, knowledge_base_id, folder_path
    FROM knowledges
    WHERE deleted_at IS NULL AND folder_path <> ''
), split AS (
    SELECT tenant_id, knowledge_base_id, string_to_array(folder_path, '/') AS all_parts, 1 AS depth
    FROM paths
    UNION ALL
    SELECT tenant_id, knowledge_base_id, all_parts, depth + 1
    FROM split
    WHERE depth < array_length(all_parts, 1)
)
INSERT INTO knowledge_folders (id, tenant_id, kb_id, parent_id, name, path, depth, is_root, created_by)
SELECT
    md5(knowledge_base_id || ':' || array_to_string(all_parts[1:depth], '/')),
    tenant_id,
    knowledge_base_id,
    CASE
        WHEN depth = 1 THEN 'folder-root-' || knowledge_base_id
        ELSE md5(knowledge_base_id || ':' || array_to_string(all_parts[1:depth - 1], '/'))
    END,
    all_parts[depth],
    array_to_string(all_parts[1:depth], '/'),
    depth,
    FALSE,
    ''
FROM split
ON CONFLICT DO NOTHING;

ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS folder_id VARCHAR(64) NOT NULL DEFAULT '';

UPDATE knowledges k
SET folder_id = COALESCE(
    (
        SELECT f.id
        FROM knowledge_folders f
        WHERE f.kb_id = k.knowledge_base_id
          AND f.path = k.folder_path
          AND f.deleted_at IS NULL
        LIMIT 1
    ),
    'folder-root-' || k.knowledge_base_id
)
WHERE k.folder_id = '' OR k.folder_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_knowledges_folder_id
    ON knowledges (tenant_id, knowledge_base_id, folder_id);
