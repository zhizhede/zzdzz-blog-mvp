-- 0007_article_autosave.sql
-- 给 articles 加 last_autosaved_at 列, 区分「主动保存(updated_at)」与「自动保存」
-- 列表排序仍按 updated_at DESC, 自动保存不打草稿债

ALTER TABLE articles
    ADD COLUMN IF NOT EXISTS last_autosaved_at TIMESTAMPTZ;

-- 部分索引: 仅 draft 走这个索引, 草稿清理任务按 last_autosaved_at 找过期 draft
-- 不建完整索引, 因为 99% 的文章不是 draft, 索引体积不划算
CREATE INDEX IF NOT EXISTS idx_articles_draft_autosaved
    ON articles (last_autosaved_at)
    WHERE visibility = 'draft';
