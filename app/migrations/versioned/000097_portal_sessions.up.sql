-- Migration: 000097_portal_sessions
-- Description: Consumer portal conversation storage (独立于内部会话,§2.13 P3)
DO $$ BEGIN RAISE NOTICE '[Migration 000097] Creating portal_sessions / portal_messages'; END $$;

CREATE TABLE IF NOT EXISTS portal_sessions (
    id            VARCHAR(36) PRIMARY KEY,
    token_kind    VARCHAR(16) NOT NULL DEFAULT 'anonymous',
    identity_key  VARCHAR(128),
    kb_ids        JSONB NOT NULL DEFAULT '[]',
    first_question TEXT,
    message_count INT NOT NULL DEFAULT 0,
    last_message_at TIMESTAMP,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN portal_sessions.token_kind IS 'anonymous/consumer';
COMMENT ON COLUMN portal_sessions.identity_key IS '脱敏身份标识(手机号哈希/匿名设备id,匿名会话为空)';
COMMENT ON COLUMN portal_sessions.kb_ids IS '本次会话可检索的 public KB 白名单';
COMMENT ON COLUMN portal_sessions.first_question IS '首问(列表页预览)';

CREATE INDEX IF NOT EXISTS idx_portal_sessions_identity ON portal_sessions (identity_key);
CREATE INDEX IF NOT EXISTS idx_portal_sessions_created ON portal_sessions (created_at DESC);

CREATE TABLE IF NOT EXISTS portal_messages (
    id          BIGSERIAL PRIMARY KEY,
    session_id  VARCHAR(36) NOT NULL,
    role        VARCHAR(16) NOT NULL,
    content     TEXT NOT NULL,
    refs        JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_portal_msg_session FOREIGN KEY (session_id) REFERENCES portal_sessions(id) ON DELETE CASCADE
);

COMMENT ON COLUMN portal_messages.role IS 'user/assistant';
COMMENT ON COLUMN portal_messages.refs IS '脱敏引用(仅标题)';

CREATE INDEX IF NOT EXISTS idx_portal_messages_session ON portal_messages (session_id, id);
