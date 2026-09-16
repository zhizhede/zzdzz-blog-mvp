-- 0015_user_password_hint.sql
-- 新增「密码提示」: 注册时用户可选填一段自由文本, 忘记密码时可通过
-- POST /api/v1/auth/password-hint 按用户名查看.
--
-- 注意: 提示是公开可读的(忘记密码场景发生在未登录态), 用户不存在时接口
-- 也返回空串, 不区分"用户不存在"和"未设置提示", 避免泄露账号存在性.
-- 列保持 NULL 可空, 存量用户一律视为未设置.

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hint VARCHAR(255);
