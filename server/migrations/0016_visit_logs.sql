-- 0016_visit_logs.sql
-- 访问日志: 记录每个到站访客的 IP, 已登录则附带 user_id, 匿名留 NULL.
-- 写入方是全局中间件 handler.RecordVisit(0016), 取真实 IP 依赖 router 里的
-- SetTrustedProxies(127.0.0.1): 宝塔 nginx 本机反代透传 X-Forwarded-For,
-- 不设可信代理时客户端可伪造该头冒充任意 IP.
--
-- 粒度: 每位访客(IP)每天至多一条, 应用侧先查当天是否已有该 IP 再插入;
-- 并发竞态下极小概率重复一条, 无害, 不为此加唯一约束.
--
-- 追加型流水表(0012 约定, 同 article_versions): 只增不改, 不带 updated_at /
-- deleted_at; user_id 不加外键, 访客记录不随用户删改联动.

CREATE TABLE IF NOT EXISTS visit_logs (
    id         BIGSERIAL PRIMARY KEY,
    ip         VARCHAR(45) NOT NULL,
    user_id    BIGINT,
    path       VARCHAR(512) NOT NULL DEFAULT '',
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_visit_logs_ip_created ON visit_logs (ip, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_visit_logs_user ON visit_logs (user_id);
