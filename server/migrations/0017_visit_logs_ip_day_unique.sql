-- 0017_visit_logs_ip_day_unique.sql
-- 访问日志同 IP 同日唯一化: 修复"先查后插"去重在并发下的竞态.
-- 打开页面时浏览器并行发出多个请求, 多个去重检查同时看到"今天还没记"而各插一条
-- (实测单日 6 个 IP 产生 52 条). 改为表达式唯一索引 + 应用侧 INSERT ON CONFLICT 兜底.
--
-- 时区必须钉死 Asia/Shanghai: 裸 created_at::date 的结果随会话时区浮动,
-- 唯一索引表达式要求确定性; 应用侧"当天"边界也按 +08:00 计算.

-- 清理存量重复行(每组同 IP 同日保留最早一条)
DELETE FROM visit_logs a
USING visit_logs b
WHERE a.ip = b.ip
  AND (a.created_at AT TIME ZONE 'Asia/Shanghai')::date
    = (b.created_at AT TIME ZONE 'Asia/Shanghai')::date
  AND a.id > b.id;

CREATE UNIQUE INDEX IF NOT EXISTS uq_visit_logs_ip_day
  ON visit_logs (ip, ((created_at AT TIME ZONE 'Asia/Shanghai')::date));
