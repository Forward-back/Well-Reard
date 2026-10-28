-- Migration: assistant 消息增加检索过程时间线（YLSW stage 事件结构化日志）
-- Description: 存检索各阶段（召回/融合/重排/上下文/生成）及相对耗时，
-- 前端刷新/重载历史后仍可展开「检索过程」。空数组 = 无检索过程。
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS rag_stage_timeline JSONB NOT NULL DEFAULT '[]'::jsonb;
