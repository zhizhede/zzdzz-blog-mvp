-- 0001_init.sql
-- zzdzz_blog MVP schema

CREATE TABLE IF NOT EXISTS users (
    id           BIGSERIAL PRIMARY KEY,
    username     VARCHAR(64)  NOT NULL UNIQUE,
    password_hash VARCHAR(128) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(64) NOT NULL UNIQUE,
    slug       VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS articles (
    id          BIGSERIAL PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    slug        VARCHAR(255),
    summary     VARCHAR(500),
    content     TEXT         NOT NULL,
    category_id BIGINT       NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    view_count  INTEGER      NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_articles_title        ON articles (title);
CREATE INDEX IF NOT EXISTS idx_articles_slug         ON articles (slug);
CREATE INDEX IF NOT EXISTS idx_articles_category_id  ON articles (category_id);
CREATE INDEX IF NOT EXISTS idx_articles_created_at   ON articles (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_categories_slug       ON categories(slug);

-- seed admin (username=admin, password=123456)
INSERT INTO users (username, password_hash)
VALUES ('admin', '$2a$10$bAojo1CnuMUuA1Oihvl1UehBQKj2v0C.pwnXYnSk4Cqw6uPpIx4Nm')
ON CONFLICT (username) DO NOTHING;

-- seed default categories
INSERT INTO categories (name, slug) VALUES
    ('技术', 'tech'),
    ('生活', 'life'),
    ('随想', 'thoughts')
ON CONFLICT (name) DO NOTHING;