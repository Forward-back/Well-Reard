-- Migration: 000094_users_is_employee
-- Description: Add is_employee flag to users (银联商务融合项目: 工号校验通过的员工)
DO $$ BEGIN RAISE NOTICE '[Migration 000094] Adding is_employee column to users'; END $$;

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_employee BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN users.is_employee IS '银联商务员工标识（注册时工号校验通过）';

DO $$ BEGIN RAISE NOTICE '[Migration 000094] is_employee column added successfully'; END $$;
