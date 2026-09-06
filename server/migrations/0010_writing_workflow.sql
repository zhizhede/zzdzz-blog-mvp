-- 0010: AI 写作工作流(语料 → 成稿), 设计见 doc/v0.4-tech-design.md
-- style_profiles: 风格卡, 每个用户一行; M2 启用提炼与注入, 表随 M1 先建.
-- article_versions: 文章版本快照, AI 采用前 / 回滚前自动写入; M3 启用接口, 表随 M1 先建.

CREATE TABLE IF NOT EXISTS style_profiles (
    user_id    BIGINT PRIMARY KEY,
    profile    TEXT NOT NULL,
    source     VARCHAR(16) NOT NULL DEFAULT 'auto',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS article_versions (
    id         BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    summary    TEXT NOT NULL DEFAULT '',
    content    TEXT NOT NULL,
    origin     VARCHAR(16) NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_article_versions_article
    ON article_versions (article_id, created_at DESC);
