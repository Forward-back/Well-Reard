-- Migration: 000037_gaze_and_indexing
-- Description: Gaze feature schema (gaze_pages table + gaze_config column + gaze_page_issues table)
-- and the indexing_strategy column on knowledge_bases.
DO $$ BEGIN RAISE NOTICE '[Migration 000037] Applying gaze + indexing schema'; END $$;

-- ---------------------------------------------------------------------------
-- 1) gaze_pages table and gaze_config column
-- Gaze pages are LLM-generated, interlinked markdown documents that form a
-- persistent gaze for a knowledge base.
-- ---------------------------------------------------------------------------
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS gaze_config JSONB;

COMMENT ON COLUMN knowledge_bases.gaze_config IS 'Gaze configuration: {"auto_ingest": bool, "synthesis_model_id": string, "gaze_language": string, "max_pages_per_ingest": int}';

CREATE TABLE IF NOT EXISTS gaze_pages (
    id              VARCHAR(36) PRIMARY KEY,
    tenant_id       BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    slug            VARCHAR(255) NOT NULL,
    title           VARCHAR(512) NOT NULL DEFAULT '',
    page_type       VARCHAR(32) NOT NULL DEFAULT 'summary',
    status          VARCHAR(32) NOT NULL DEFAULT 'published',
    content         TEXT NOT NULL DEFAULT '',
    summary         TEXT NOT NULL DEFAULT '',
    parent_slug     VARCHAR(255) NOT NULL DEFAULT '',
    -- folder_id is the single source of truth for a page's placement in the
    -- directory tree (FK to gaze_folders.id; '' = gaze root). category_path /
    -- gaze_path / depth below are denormalized projections of the folder chain,
    -- kept in sync on write so list/index/search queries need no folder join.
    folder_id       VARCHAR(36) NOT NULL DEFAULT '',
    category_path   JSONB DEFAULT '[]'::JSONB,
    gaze_path       VARCHAR(1024) NOT NULL DEFAULT '',
    depth           INT NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    source_refs     JSONB DEFAULT '[]'::JSONB,
    chunk_refs      JSONB DEFAULT '[]'::JSONB,
    in_links        JSONB DEFAULT '[]'::JSONB,
    out_links       JSONB DEFAULT '[]'::JSONB,
    page_metadata   JSONB DEFAULT '{}'::JSONB,
    aliases         JSONB DEFAULT '[]'::JSONB,
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE
);

-- slug must be unique within a knowledge base (for non-deleted pages)
CREATE UNIQUE INDEX IF NOT EXISTS idx_gaze_pages_kb_slug
    ON gaze_pages (knowledge_base_id, slug)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_gaze_pages_kb_id
    ON gaze_pages (knowledge_base_id);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_page_type
    ON gaze_pages (knowledge_base_id, page_type);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_parent_slug
    ON gaze_pages (knowledge_base_id, parent_slug);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_tree
    ON gaze_pages (knowledge_base_id, page_type, gaze_path, sort_order, title);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_folder
    ON gaze_pages (knowledge_base_id, folder_id);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_tenant_id
    ON gaze_pages (tenant_id);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_deleted_at
    ON gaze_pages (deleted_at);

CREATE INDEX IF NOT EXISTS idx_gaze_pages_fulltext
    ON gaze_pages USING GIN (to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(content, '')));

-- ---------------------------------------------------------------------------
-- 1b) gaze_folders table
-- First-class directory nodes for the gaze browser. A folder exists
-- independently of any page, so empty folders persist (the user can build a
-- skeleton and file pages into it later). parent_id forms an adjacency-list
-- tree ('' = root); path is the materialized "/"-joined name chain kept for
-- cheap display/sort. gaze_pages.folder_id references id; renaming/moving a
-- folder updates this row's subtree and the affected pages' cached path.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gaze_folders (
    id                VARCHAR(36) PRIMARY KEY,
    tenant_id         BIGINT NOT NULL DEFAULT 0,
    knowledge_base_id VARCHAR(36) NOT NULL,
    parent_id         VARCHAR(36) NOT NULL DEFAULT '',
    name              VARCHAR(255) NOT NULL,
    path              VARCHAR(1024) NOT NULL DEFAULT '',
    depth             INT NOT NULL DEFAULT 0,
    sort_order        INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMP WITH TIME ZONE
);

-- A folder name is unique among its live siblings under the same parent.
CREATE UNIQUE INDEX IF NOT EXISTS idx_gaze_folders_parent_name
    ON gaze_folders (knowledge_base_id, parent_id, name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_gaze_folders_parent
    ON gaze_folders (knowledge_base_id, parent_id);

CREATE INDEX IF NOT EXISTS idx_gaze_folders_deleted_at
    ON gaze_folders (deleted_at);

-- ---------------------------------------------------------------------------
-- 2) gaze_page_issues table
-- Reports issues against generated gaze pages (LLM-flagged or user-flagged).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gaze_page_issues (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    issue_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    suspected_knowledge_ids JSONB,
    status VARCHAR(20) DEFAULT 'pending' NOT NULL,
    reported_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_gaze_page_issues_tenant_id ON gaze_page_issues(tenant_id);
CREATE INDEX IF NOT EXISTS idx_gaze_page_issues_knowledge_base_id ON gaze_page_issues(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_gaze_page_issues_slug ON gaze_page_issues(slug);
CREATE INDEX IF NOT EXISTS idx_gaze_page_issues_status ON gaze_page_issues(status);

-- ---------------------------------------------------------------------------
-- 3) indexing_strategy column on knowledge_bases
-- Controls which indexing pipelines are active (vector, keyword, gaze, graph).
-- Backfill: existing rows get vector+keyword=true (legacy default behavior);
-- gaze_enabled / graph_enabled stay false until explicitly enabled.
-- ---------------------------------------------------------------------------
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS indexing_strategy JSONB;

COMMENT ON COLUMN knowledge_bases.indexing_strategy IS 'Indexing pipelines strategy: {"vector_enabled": bool, "keyword_enabled": bool, "gaze_enabled": bool, "graph_enabled": bool}';

UPDATE knowledge_bases
SET indexing_strategy = jsonb_build_object(
    'vector_enabled',  TRUE,
    'keyword_enabled', TRUE,
    'gaze_enabled',    FALSE,
    'graph_enabled',   FALSE
)
WHERE indexing_strategy IS NULL;

DO $$ BEGIN RAISE NOTICE '[Migration 000037] gaze + indexing schema applied successfully'; END $$;
