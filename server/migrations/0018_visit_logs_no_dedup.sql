-- 0018_visit_logs_no_dedup.sql
-- 应用户要求放宽为无差别记录: 移除 0017 引入的同 IP 同日唯一索引,
-- 每个通过噪音过滤的请求各记一条, 归属随请求自身的登录态
-- (页面加载无 token 记匿名, 登录后的 API 请求带 user_id).
--
-- 应用侧同步移除 ON CONFLICT 逻辑, 纯插入.

DROP INDEX IF EXISTS uq_visit_logs_ip_day;
