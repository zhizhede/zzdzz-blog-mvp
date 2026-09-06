-- 0012_soft_delete.sql
-- 约定变更(2026-09-07): 引入软删除. 数据安全优先, 删除必须可恢复.
-- 此前全库无 deleted_at(0001-0011), 物理删除; 本迁移起实体表删除转为标记删除.
--
-- 应用侧联动(GORM gorm.DeletedAt):
--   - db.Delete 自动变为 UPDATE deleted_at
--   - 所有 GORM 查询自动过滤已删行, 无需业务代码逐处加条件
--   - draft-cleanup 为垃圾回收语义, 保持原生 SQL 硬删, 但只清未删行
--
-- 范围: 实体表(有独立生命周期、用户可感知的数据)加 deleted_at.
-- 以下三张表有意不加, 加了会出 bug:
--   article_tags    纯关联表, 生命周期跟随两端; replaceArticleTags 是硬删+重建,
--                   混入软删行会产生幽灵关联
--   ai_chunks       派生索引, 生命周期由 removeRecall 应用层管理; 加列后
--                   UNIQUE(source_type, source_id, chunk_index) 会让重嵌入报冲突
--   article_versions 追加型审计备份, prune(保留50版)是它的生命周期;
--                   软删会让 prune 失效, 备份无限膨胀, 且可篡改的历史违背备份语义
--
-- 唯一约束必须同步改为「仅对未删除行生效」的部分唯一索引,
-- 否则软删「技术」分类后永远无法再创建同名分类.

ALTER TABLE users            ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE categories       ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE articles         ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE tags             ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE ai_conversations ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE ai_messages      ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE style_profiles   ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 内联 UNIQUE 的默认约束名: <表>_<列>_key
ALTER TABLE users      DROP CONSTRAINT IF EXISTS users_username_key;
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_name_key;
ALTER TABLE tags       DROP CONSTRAINT IF EXISTS tags_name_key;
ALTER TABLE tags       DROP CONSTRAINT IF EXISTS tags_slug_key;

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username_alive  ON users(username)  WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_categories_name_alive ON categories(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tags_name_alive       ON tags(name)       WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tags_slug_alive       ON tags(slug)       WHERE deleted_at IS NULL;
