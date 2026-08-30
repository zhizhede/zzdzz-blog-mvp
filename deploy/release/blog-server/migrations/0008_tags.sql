-- 0008_tags.sql
-- 标签系统: tags + article_tags (多对多中间表)
-- 老文章没标签, 不做 backfill, 由作者自己挑要不要打

CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    slug VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS article_tags (
    article_id BIGINT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES tags(id)     ON DELETE CASCADE,
    PRIMARY KEY (article_id, tag_id)
);

-- 反向查「某标签下的所有文章」走索引
CREATE INDEX IF NOT EXISTS idx_article_tags_tag ON article_tags (tag_id);
