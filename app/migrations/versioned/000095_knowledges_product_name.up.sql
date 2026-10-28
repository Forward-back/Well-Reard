-- Migration: 000095_knowledges_product_name
-- Description: Add product_name column to knowledges (UMS 域元数据，chunks 冗余来源)
DO $$ BEGIN RAISE NOTICE '[Migration 000095] Adding product_name column to knowledges'; END $$;

ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS product_name VARCHAR(100);

COMMENT ON COLUMN knowledges.product_name IS 'UMS 产品名（如 全民付蓝鲸）';

DO $$ BEGIN RAISE NOTICE '[Migration 000095] product_name column added successfully'; END $$;
