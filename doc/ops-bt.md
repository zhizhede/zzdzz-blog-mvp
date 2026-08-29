# 线上运维排障手册 (zzdzz-blog / 宝塔)

> 单机部署：1 台宝塔 Linux，PG + Go 后端 + Vue 前端 + Nginx 反代，无域名。
>
> 配套：`doc/deploy-bt.md`(首次部署流程) / `doc/incidents/2026-08-29-ai-streaming.md`(事故复盘模板)。
>
> 本文档只列**路径 + 入口 + 几条命令**，方便出问题时不靠记忆直接找到。

## 一、宝塔面板

| 项 | 值 / 命令 |
|---|---|
| 面板地址 | `https://<host>:26152/<入口路径>` |
| 入口路径 | 通过 SSH 上服务器后 `bt 14` 查看 |
| 用户名 | `bt 14` 查 |
| 密码 | SSH 上 `bt 5` 修改;**初始密码只在首次 `bt 14` 前能正确获取** |
| 公网安全组 | 需放行 `<面板端口>/TCP`(默认可能是 8888,改过是别的) |

查 / 改密码:

```bash
ssh root@<host>
bt 14   # 查看入口 + 用户名 + 初始密码
bt 5    # 修改面板密码
```

## 二、SSH 登录服务器

| 项 | 路径 |
|---|---|
| 本地私钥 | 仓库 `keys/<host>_id_ed25519`(已 gitignore) |
| 用户 | `root`(默认) |
| 端口 | 22 |

```bash
ssh -i keys/<host>_id_ed25519 root@<host>
```

> 私钥路径在 `.gitignore`,**不要 commit**。新机器加密钥时另说。

## 三、前后端文件路径

### 后端 (Go)

| 路径 | 内容 |
|---|---|
| `/www/wwwroot/blog-server/blog-server` | 当前运行的二进制(28MB) |
| `/www/wwwroot/blog-server/web/` | Go 后端 serve 的前端 dist(同 nginx dist 的内容) |
| `/www/wwwroot/blog-server/config/config.local.yaml` | 真实配置(JWT secret / DB 密码 / AI key) |
| `/www/wwwroot/blog-server.bak-YYYYMMDD-HHMMSS/` | 历史二进制备份(滚动保留,自动产生) |

进程 / 日志:

| 路径 | 用途 |
|---|---|
| `/proc/<pid>/cwd` → `/www/wwwroot/blog-server` | 进程工作目录 |
| `/www/wwwlogs/go/blog_server.log` | **GORM SQL 日志**(每次 UPDATE 一行,排查 DB 写入问题查这里) |
| `journalctl -u blog_server` 或 `/www/server/panel/logs/` | 宝塔项目 stderr/stdout(启动失败看这里) |

### 前端 (Vue dist)

| 路径 | 谁在 serve |
|---|---|
| `/www/wwwroot/blog-ui/dist/` | **nginx** 反代站点 |
| `/www/wwwroot/blog-server/web/` | **Go 后端** `r.Static` + `c.File` 直 serve(同份内容必须同步) |

详见 `doc/deploy-bt.md` §九 两处同步。

### nginx

| 路径 | 用途 |
|---|---|
| `/www/server/panel/vhost/nginx/101.126.22.219.conf` | **线上站点配置**(宝塔管理,不是仓库里的 `web/nginx.conf`) |
| `/www/server/panel/vhost/nginx/*.conf.bak-*` | 我每次改 nginx 前的自动备份 |
| `/www/server/nginx/conf/nginx.conf` | nginx 顶层配置(含 `gzip on` 默认值,改 SSE 路径要按 `location` 局部关) |
| `/www/wwwlogs/101.126.22.219.log` | nginx access log(查请求 URL + 状态码) |
| `/www/wwwlogs/101.126.22.219.error.log` | nginx error log(502/404 看这里) |

### PostgreSQL(docker)

| 项 | 值 |
|---|---|
| 监听端口 | `35432`(暴露在 host) |
| 容器 | `zzdzz-pg` |
| 镜像 | `postgres:16-alpine` |
| 数据卷 | `pg_data` |

连:

```bash
docker exec -it zzdzz-pg psql -U postgres -d zzdzz_blog
# 或从 host
PGPASSWORD='<password>' psql -h 127.0.0.1 -p 35432 -U postgres -d zzdzz_blog
```

## 四、几条诊断命令

```bash
# nginx 改完先 -t,再 reload(不中断现有连接)
nginx -t && nginx -s reload

# 看 nginx 站点监听 + 谁在用
ss -tlnp | grep -E ':(80|8080|35432)\s'

# 看 backend 进程
ps -ef | grep -E "blog-server|nginx:"

# 抓 backend → nginx 字节流(验证 SSE 是否真流)
tcpdump -ni lo -tttt "tcp port 8080"

# 抓 nginx → 浏览器字节流
tcpdump -ni eth0 -tttt "tcp port 80"

# 看后端 GORM 日志最近 delta 写入频率
tail -100 /www/wwwlogs/go/blog_server.log

# 看 nginx access log 最近 AI 请求
tail -50 /www/wwwlogs/101.126.22.219.log | grep ai
```

字节层排查清单见 `doc/incidents/2026-08-29-ai-streaming.md` §3。