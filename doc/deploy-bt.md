# 宝塔部署备忘 (zzdzz-blog)

> 单机部署：1 台宝塔 Linux，PG + Go 后端 + Vue 前端 + Nginx 反代，**无域名**。

## 一、产物结构 (`deploy/release/blog-server/`)

```
blog-server/
├── blog-server              Linux 静态二进制 (28MB, ELF, no deps)
├── web/                     前端构建产物 (index.html + assets/)
├── config/
│   └── config.production.yaml   生产配置 (gitignore)
├── migrations/
│   └── 0001_init.sql            建表 + 种子数据
├── init.sh                   [可选] 一键初始化 (schema + JWT + 改密码)
├── start.sh / stop.sh        备用启停
└── README.md                 宝塔面板填写说明
```

## 二、编译命令

```bash
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/server-linux-amd64 .
cp bin/server-linux-amd64 ../deploy/release/blog-server/blog-server
cp -r ../web/dist ../deploy/release/blog-server/web
```

## 三、上传服务器

```bash
scp deploy/release/blog-server/blog-server       root@<host>:/www/wwwroot/blog-server/blog-server
scp -r deploy/release/blog-server/web            root@<host>:/www/wwwroot/blog-server/web
# 首次部署再传 config + migrations + *.sh
```

## 四、宝塔面板

### Go 项目
| 字段 | 值 |
|---|---|
| 项目执行文件 | `/www/wwwroot/blog-server/blog-server` |
| 项目名称 | `blog_server` (正则 `^[0-9A-Za-z_]$`，连字符禁用) |
| 项目端口 | `8080` |
| 执行命令 | 留空 |
| 运行用户 | `www` |
| 开机启动 | 勾 |

### Nginx 网站
- 添加站点 → 域名填 `<host>`（无域名填 IP）→ PHP 版本选"纯静态"
- 设置 → 反向代理 → 目标 URL `http://127.0.0.1:8080`，发送域名 `$http_host`
- 配置文件 → 反代 `location /` 段**整段替换**为：

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Upgrade           $http_upgrade;
    proxy_set_header Connection        "upgrade";
    proxy_connect_timeout 30s;
    proxy_send_timeout    60s;
    proxy_read_timeout    60s;
    client_max_body_size  20m;
}
```

## 五、数据库初始化

```bash
PGPASSWORD='<pg_pass>' psql \
  -h 127.0.0.1 -p 35432 -U postgres -d zzdzz_blog \
  -v ON_ERROR_STOP=1 \
  -f migrations/0001_init.sql
```

幂等，可重跑。种子：管理员 `admin/123456`、分类 技术/生活/随想。

## 六、上线后必须做（安全收尾）

1. **替换 JWT secret**（占位符 `REPLACE_WITH_OPENSSL_RAND_HEX_32_OUTPUT` 在 git/对话/部署包里都是公开的）：
   ```bash
   NEW=$(openssl rand -hex 32)
   sed -i "s|REPLACE_WITH_OPENSSL_RAND_HEX_32_OUTPUT|$NEW|" config/config.production.yaml
   # 宝塔面板 → 重启项目
   ```
2. **改 admin 默认密码** `admin/123456`（写在 SQL 注释里，公开）：
   ```bash
   PGPASSWORD='<pg_pass>' psql -h 127.0.0.1 -p 35432 -U postgres -d zzdzz_blog \
     -c "UPDATE users SET password_hash = crypt('<新密码>', gen_salt('bf', 10)) WHERE username='admin';"
   ```
3. **关掉 8080 公网放行**（仅 80/Nginx 暴露）：宝塔"安全"和云厂商安全组各删一条 8080/TCP。

## 七、配置优先级（main.go 查找顺序）

```
$ZZDZZ_CONFIG (环境变量)
config/config.production.yaml
config/config.local.yaml
config/config.yaml
```

约定：
- `config.yaml` 入库（开发模板，占位符）
- `config.local.yaml` / `config.production.yaml` gitignore（真实值）

## 八、文件清单变化历史

| 改动 | 原因 |
|---|---|
| `main.go` 加 `config.production.yaml` 候选 | 生产配置不被查找，原代码只认 `local` |
| `router.go` 加 `r.Static("/assets", ...)` + `c.File("./web/index.html")` + `r.NoRoute` | 根路径返回 SPA，让前端路由接管 |
| `.gitignore` 加 `*.production.yaml / *.staging.yaml` 通配 | 防止未来再误提交真实密钥 |
