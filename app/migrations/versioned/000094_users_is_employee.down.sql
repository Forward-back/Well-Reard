-- Migration: 000094_users_is_employee (down)
-- Description: Remove is_employee column from users.
DO $$ BEGIN RAISE NOTICE '[Migration 000094 down] Removing is_employee column from users'; END $$;

ALTER TABLE users DROP COLUMN IF EXISTS is_employee;

DO $$ BEGIN RAISE NOTICE '[Migration 000094 down] is_employee column removed successfully'; END $$;
