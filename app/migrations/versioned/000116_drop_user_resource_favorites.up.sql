-- Migration: 000116_drop_user_resource_favorites
-- The favorites subsystem was removed; its table is unused and empty.
DO $$ BEGIN RAISE NOTICE '[Migration 000116] Dropping table: user_resource_favorites'; END $$;

DROP INDEX IF EXISTS idx_user_resource_favorites_tenant_id;
DROP INDEX IF EXISTS idx_user_resource_favorites_user_tenant_type_created_at;
DROP TABLE IF EXISTS user_resource_favorites;

DO $$ BEGIN RAISE NOTICE '[Migration 000116] user_resource_favorites table dropped'; END $$;
