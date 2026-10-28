-- Migration: user 消息记录指代消解的展开信息（需求五 UI 返工）
-- Description: 保存原文/改写文的实体映射、上轮回答锚点、must_keep 校验与耗时，
-- 供刷新历史后展开「改写已完成」卡片使用。
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS query_rewrite_meta JSONB;
