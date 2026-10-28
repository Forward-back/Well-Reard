-- Migration: user 消息记录指代消解后的实际检索 query（需求五）
-- Description: 空字符串表示本轮未改写或改写失败放行；非空时前端在
-- 用户消息下展示「已改写为」，历史刷新后仍保留。
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS query_rewritten TEXT NOT NULL DEFAULT '';
