-- 0011_referential_integrity.sql
-- 补齐引用完整性. (注: "全库无软删除"是 0011 时的约定, 0012 起实体表已引入
-- 软删除; 本迁移的 FK 级联依然有效且更有必要——级联在物理删除时触发,
-- 覆盖 draft-cleanup 的硬删与未来的回收站彻底清除路径.)
--
-- 缺口盘点(0001-0010 逐表核对):
--   article_tags      双 CASCADE(0008)            ✓ DB 层已覆盖
--   ai_messages       CASCADE(0002)               ✓ DB 层已覆盖
--   ai_chunks         多态表(source_type 预留), 不加 FK,
--                     删除时由 ArticleService.removeRecall 应用层清理 ✓
--   article_versions  article_id 无 FK            ✗ 删文章/清理草稿留孤儿版本行
--   style_profiles    user_id 无 FK               ✗ 用户应用层永不删, 此为兜底
--
-- 幂等: 可重复执行. 重复执行时先重复清一次孤儿, 再重建同名约束.

-- 1) 清理历史孤儿版本行(所属文章已物理删除, 这些行不可达, 属于垃圾数据)
DELETE FROM article_versions v
WHERE NOT EXISTS (SELECT 1 FROM articles a WHERE a.id = v.article_id);

-- 2) 版本随文章级联删除
ALTER TABLE article_versions DROP CONSTRAINT IF EXISTS fk_article_versions_article;
ALTER TABLE article_versions
    ADD CONSTRAINT fk_article_versions_article
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE;

-- 3) 风格卡随用户级联删除(兜底: 防手工删用户时留孤儿)
ALTER TABLE style_profiles DROP CONSTRAINT IF EXISTS fk_style_profiles_user;
ALTER TABLE style_profiles
    ADD CONSTRAINT fk_style_profiles_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
