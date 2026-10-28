-- Migration: 000093_ums_domain_metadata
-- Description: Add UMS (银联商务) domain metadata columns for the YLSW retrieval engine.
-- knowledges: document-level fields extracted by docreader (文号/机构/日期/类型)
-- chunks: chunk-level fields used by metadata search and org-scope filtering
DO $$ BEGIN RAISE NOTICE '[Migration 000093] Adding UMS domain metadata columns'; END $$;

ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS doc_type VARCHAR(50);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS org_name VARCHAR(255);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS doc_number VARCHAR(100);
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS doc_date VARCHAR(50);

ALTER TABLE chunks ADD COLUMN IF NOT EXISTS doc_type VARCHAR(50);
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS product_name VARCHAR(100);
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS org_name VARCHAR(255);

COMMENT ON COLUMN knowledges.doc_type IS 'UMS 文档类型: regulation/manual/report/news/table/general';
COMMENT ON COLUMN knowledges.org_name IS 'UMS 机构规范名（如 中国人民银行/银联商务）';
COMMENT ON COLUMN knowledges.doc_number IS 'UMS 文号（如 银发2015325号）';
COMMENT ON COLUMN knowledges.doc_date IS 'UMS 文档日期';
COMMENT ON COLUMN chunks.doc_type IS 'UMS 文档类型（冗余自 knowledges，供元数据检索）';
COMMENT ON COLUMN chunks.product_name IS 'UMS 产品名（如 全民付蓝鲸）';
COMMENT ON COLUMN chunks.org_name IS 'UMS 机构规范名（冗余自 knowledges，供机构范围过滤）';

CREATE INDEX IF NOT EXISTS idx_chunks_product_name ON chunks(product_name);
CREATE INDEX IF NOT EXISTS idx_chunks_doc_type ON chunks(doc_type);
CREATE INDEX IF NOT EXISTS idx_chunks_org_name ON chunks(org_name);
CREATE INDEX IF NOT EXISTS idx_knowledges_doc_number ON knowledges(doc_number);

DO $$ BEGIN RAISE NOTICE '[Migration 000093] UMS domain metadata columns added successfully'; END $$;
