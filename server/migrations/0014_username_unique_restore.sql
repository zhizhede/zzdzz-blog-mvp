-- 0014_username_unique_restore.sql
-- 约定变更(2026-09-08): 撤销 0013 的「用户名可重名」, 用户名恢复全系统唯一.
-- 原因: 重名靠密码区分在「同名同密码」时无解——登录恒命中 id 较小者,
-- 另一同名账号永远无法登录; 产品语义上重名得不偿失.
--
-- 保留的部分(0013 引入): uuid 机制原样保留.
--   uuid 仍是用户的稳定唯一身份标识, 对外接口(/auth/login /auth/register
--   /auth/me, 用户管理 List)继续暴露; 区别只在登录/注册的判定键回到 username.
--
-- 应用前人工校验(必须无结果才可执行本迁移):
--   SELECT username, COUNT(*) FROM users WHERE deleted_at IS NULL
--   GROUP BY username HAVING COUNT(*) > 1;

DROP INDEX IF EXISTS idx_users_username_alive;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username_alive ON users(username) WHERE deleted_at IS NULL;
