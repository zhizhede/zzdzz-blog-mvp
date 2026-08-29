-- 0006_articles_claim_to_admin.sql
-- 把现有所有 author_id IS NULL 的老文章归属到 admin(id=1).
-- 决策: 老文章保持无作者会让 admin / 非 admin 都能改, 体验割裂, 直接 backfill 到 admin 更顺手.
-- 注意: admin 用户 id 写死 1, 不引用 username, 避免 admin 被重命名时连带出错.

UPDATE articles SET author_id = 1 WHERE author_id IS NULL;