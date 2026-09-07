-- 0013_user_uuid_open_registration.sql
-- 约定变更(2026-09-07): 用户身份改由 uuid 唯一标识, username 放开重名.
-- 背景博客首页对游客开放后, 开放注册随即上线: 游客输用户名 + 密码即可注册,
-- 用户名允许重名(同名用户靠密码区分), 因此 username 不再是身份标识.
--
-- 应用侧联动(GORM model.User):
--   - 新增 UUID 列, 注册/建号时由应用生成(google/uuid v4, 36 位带连字符)
--   - 登录按 username 查出全部同名行, 逐个 bcrypt 比对密码
--   - 对外接口(/auth/login /auth/me /auth/register, 用户管理 List)一律带 uuid
--
-- 安全约束: 保留用户名不再唯一, 但 ZZDZZ_ADMIN_USERNAMES 命中的用户名
-- (默认 "admin")是保留名, 注册和管理员建号都会拒绝, 防止重名机制被用来
-- 伪造 admin 身份(is_admin 仍按用户名推断).

-- 1) 加列(先允许 NULL, 回填后再收紧)
ALTER TABLE users ADD COLUMN IF NOT EXISTS uuid VARCHAR(36);

-- 2) 存量用户回填 uuid (PG13+ 内置 gen_random_uuid)
UPDATE users SET uuid = gen_random_uuid()::text WHERE uuid IS NULL OR uuid = '';

-- 3) 收紧为 NOT NULL
ALTER TABLE users ALTER COLUMN uuid SET NOT NULL;

-- 4) uuid 是新的唯一身份: 沿用 0012 的部分唯一索引约定(仅对未删除行生效)
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_uuid_alive ON users(uuid) WHERE deleted_at IS NULL;

-- 5) username 放开重名: 撤掉 0012 的唯一索引, 降级为普通索引(登录查询仍按 username 查)
DROP INDEX IF EXISTS uq_users_username_alive;
CREATE INDEX IF NOT EXISTS idx_users_username_alive ON users(username) WHERE deleted_at IS NULL;
