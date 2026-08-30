-- 0004_user_active.sql
-- 给 users 表加 is_active 字段, 支持后台启用/禁用账户

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- 默认全部启用, 保持向后兼容 (老账户不会因为迁移而无法登录)
UPDATE users SET is_active = TRUE WHERE is_active IS NULL;