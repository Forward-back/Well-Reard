-- Migration: 000098_enterprise_metadata (down)
DO $$ BEGIN RAISE NOTICE '[Migration 000098] Dropping enterprise metadata columns'; END $$;

DROP TABLE IF EXISTS knowledge_custom_field_defs;

ALTER TABLE chunks DROP COLUMN IF EXISTS department;
ALTER TABLE chunks DROP COLUMN IF EXISTS sensitivity;
ALTER TABLE chunks DROP COLUMN IF EXISTS expiry_date;
ALTER TABLE chunks DROP COLUMN IF EXISTS effective_date;
ALTER TABLE chunks DROP COLUMN IF EXISTS lifecycle_status;

ALTER TABLE knowledges DROP COLUMN IF EXISTS department;
ALTER TABLE knowledges DROP COLUMN IF EXISTS sensitivity;
ALTER TABLE knowledges DROP COLUMN IF EXISTS superseded_by;
ALTER TABLE knowledges DROP COLUMN IF EXISTS expiry_date;
ALTER TABLE knowledges DROP COLUMN IF EXISTS effective_date;
ALTER TABLE knowledges DROP COLUMN IF EXISTS lifecycle_status;

DO $$ BEGIN RAISE NOTICE '[Migration 000098] Enterprise metadata columns dropped successfully'; END $$;
