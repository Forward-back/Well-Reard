-- Migration: 000095_knowledges_product_name (down)
-- Description: Remove product_name column from knowledges.
DO $$ BEGIN RAISE NOTICE '[Migration 000095 down] Removing product_name column from knowledges'; END $$;

ALTER TABLE knowledges DROP COLUMN IF EXISTS product_name;

DO $$ BEGIN RAISE NOTICE '[Migration 000095 down] product_name column removed successfully'; END $$;
