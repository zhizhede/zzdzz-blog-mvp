-- 0019: 快捷用语板(quick_phrases)
-- 用户自预设的常用语句, 笔记/文章编辑器点击即插入正文光标处; 严格按用户隔离.
-- 沿用 0012 软删约定(实体表带 deleted_at, GORM 查询自动过滤已删行);
-- user_id 外键级联沿用 0011 约定: 物理删除用户时短语随之清理(兜底).

CREATE TABLE IF NOT EXISTS quick_phrases (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL,
    content    VARCHAR(200) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_quick_phrases_user
    ON quick_phrases (user_id, sort_order, id);

ALTER TABLE quick_phrases DROP CONSTRAINT IF EXISTS fk_quick_phrases_user;
ALTER TABLE quick_phrases
    ADD CONSTRAINT fk_quick_phrases_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
