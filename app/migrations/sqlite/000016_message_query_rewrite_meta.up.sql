-- Migration: user 消息记录指代消解的展开信息（需求五 UI 返工）
ALTER TABLE messages ADD COLUMN query_rewrite_meta TEXT;
