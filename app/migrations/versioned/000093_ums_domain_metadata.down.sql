-- Migration: 000093_ums_domain_metadata (down)
-- Description: Remove UMS domain metadata columns.
DO $$ BEGIN RAISE NOTICE '[Migration 000093 down] Removing UMS domain metadata columns'; END $$;

DROP INDEX IF EXISTS idx_chunks_product_name;
DROP INDEX IF EXISTS idx_chunks_doc_type;
DROP INDEX IF EXISTS idx_chunks_org_name;
DROP INDEX IF EXISTS idx_knowledges_doc_number;

ALTER TABLE chunks DROP COLUMN IF EXISTS doc_type;
ALTER TABLE chunks DROP COLUMN IF EXISTS product_name;
ALTER TABLE chunks DROP COLUMN IF EXISTS org_name;

ALTER TABLE knowledges DROP COLUMN IF EXISTS doc_date;
ALTER TABLE knowledges DROP COLUMN IF EXISTS doc_number;
ALTER TABLE knowledges DROP COLUMN IF EXISTS org_name;
ALTER TABLE knowledges DROP COLUMN IF EXISTS doc_type;

DO $$ BEGIN RAISE NOTICE '[Migration 000093 down] UMS domain metadata columns removed successfully'; END $$;
