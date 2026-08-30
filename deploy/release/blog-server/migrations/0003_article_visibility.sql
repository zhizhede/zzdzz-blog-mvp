-- 0003_article_visibility.sql
-- 文章可见性: public(公开) / private(仅自己) / draft(草稿, 连 URL 都404)
-- 默认 public 保持向后兼容, 已有文章全部沿用旧行为

ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(16) NOT NULL DEFAULT 'public'
        CHECK (visibility IN ('public', 'private', 'draft'));

-- 用于公开列表过滤 (public only)
CREATE INDEX IF NOT EXISTS idx_articles_visibility_created
    ON articles (visibility, created_at DESC);