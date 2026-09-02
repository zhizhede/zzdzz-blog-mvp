-- v0.3 AI 回顾(RAG 知识召回): 向量索引表, 设计见 doc/v0.3-tech-design.md
-- 前提: Postgres 需可安装 pgvector 扩展

CREATE EXTENSION IF NOT EXISTS vector;

-- 维度 1536 与 ai.embedding.dims 一致, 启动时会校验.
-- 实测 MiniMax embo-01 返回 1536 维(官方文档写 1024 与实际不符).
-- 换 embedding 模型/维度 = 新迁移改列类型 + embed-backfill 全量重嵌.
CREATE TABLE IF NOT EXISTS ai_chunks (
  id          BIGSERIAL PRIMARY KEY,
  source_type VARCHAR(16)  NOT NULL DEFAULT 'article', -- 预留: article / message / ...
  source_id   BIGINT       NOT NULL,                   -- article.id
  owner_id    BIGINT,                                  -- 冗余 articles.author_id, NULL = 老文章
  scope       VARCHAR(16)  NOT NULL,                   -- public / private; draft 不入索引
  chunk_index INT          NOT NULL,
  heading     VARCHAR(255),                            -- 所属小节标题, 供引用展示
  content     TEXT         NOT NULL,
  metadata    JSONB,                                   -- {title, category, tags, published_at}
  embedding   vector(1536) NOT NULL,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
  UNIQUE (source_type, source_id, chunk_index)
);

-- 语料量级(数千行)下, OR 过滤走 HNSW 扫描后过滤即可, 不需要 partition
CREATE INDEX IF NOT EXISTS idx_ai_chunks_hnsw   ON ai_chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_ai_chunks_scope  ON ai_chunks (scope, owner_id);
CREATE INDEX IF NOT EXISTS idx_ai_chunks_source ON ai_chunks (source_type, source_id);
