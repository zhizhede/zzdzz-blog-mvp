-- 0002_ai.sql
-- AI 对话持久化:会话表 + 消息表

CREATE TABLE IF NOT EXISTS ai_conversations (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(100) NOT NULL DEFAULT '未命名会话',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_conv_user_updated
    ON ai_conversations (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS ai_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT       NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            VARCHAR(16)  NOT NULL CHECK (role IN ('user','assistant','system')),
    content         TEXT         NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_msg_conv_created
    ON ai_messages (conversation_id, created_at);