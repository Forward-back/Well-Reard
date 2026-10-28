-- Migration: user 消息记录指代消解后的实际检索 query（需求五）
ALTER TABLE messages ADD COLUMN query_rewritten TEXT NOT NULL DEFAULT '';
