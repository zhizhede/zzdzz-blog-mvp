-- 0005_article_author.sql
-- 给 articles 加 author_id 字段, 支持「我的笔记」按作者归属与权限隔离

ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS author_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_articles_author_created
    ON articles (author_id, created_at DESC);

-- 老文章保持 author_id = NULL: 由 handler 兜底, 仅 admin 可改/删, 非 admin 一律拒绝
-- 不做 backfill, 避免把老文章错误归属到 admin