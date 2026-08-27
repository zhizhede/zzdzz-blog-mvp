# 部署指南

## 一键部署（推荐）

### 1. 在云服务器上克隆仓库

```bash
cd /www/wwwroot
git clone https://github.com/zhizhede/zzdzz-blog-mvp.git zzdzz-blog
cd zzdzz-blog
```

### 2. 准备环境变量

```bash
cp deploy/.env.example deploy/.env
vim deploy/.env   # 修改密码、密钥、AI key
```

`JWT_SECRET` 用 `openssl rand -hex 32` 生成。

### 3. 执行部署脚本

```bash
bash deploy/deploy.sh
```

脚本会自动：
- `git pull` 拉取最新代码
- Docker 构建后端 + 前端镜像
- 启动 postgres / backend / frontend 三个容器
- 首次启动时会自动执行 `migrations/0001_init.sql`（通过 `docker-entrypoint-initdb.d`）

### 4. 验证

```bash
# 健康检查
curl http://localhost/api/v1/ping

# 看日志
docker compose -f deploy/docker-compose.yml logs -f backend

# 进 PG 容器手动查数据
docker exec -it zzdzz-pg psql -U postgres -d zzdzz_blog
```

## 架构

```
Internet (端口 80)
     │
     ▼
┌──────────────────┐
│ frontend 容器    │  nginx 服务 dist/,反代 /api/* 到 backend:8080
│ (nginx + Vite产物)│
└──────────────────┘
     │
     ├──── /api/* ──→ ┌──────────────┐
     │                │ backend 容器 │  Go 二进制
     │                │ (alpine)    │
     │                └──────────────┘
     │                       │
     └────────── (内部网络) ──┴─────→ ┌──────────────┐
                                       │ postgres 容器 │
                                       │ (PG 16-alpine)│
                                       └──────────────┘
```

## 常见操作

```bash
# 停止
docker compose -f deploy/docker-compose.yml down

# 重启某个服务
docker compose -f deploy/docker-compose.yml restart backend

# 重新构建并启动
docker compose -f deploy/docker-compose.yml up -d --build backend

# 备份数据库
docker exec zzdzz-pg pg_dump -U postgres zzdzz_blog > backup_$(date +%Y%m%d).sql

# 恢复
cat backup_20260101.sql | docker exec -i zzdzz-pg psql -U postgres -d zzdzz_blog
```

## HTTPS（可选）

云服务器上 80 端口跑前端后，套一层 HTTPS：

1. 域名解析到服务器 IP
2. 宝塔面板 → SSL → Let's Encrypt 一键签
3. 宝塔会在 443 端口反代到 80（如果前端 docker compose 映射到 80）

或者直接用 Caddy 自动续签：
```bash
docker run -d \
  --name caddy \
  -p 80:80 -p 443:443 \
  -v /path/to/Caddyfile:/etc/caddy/Caddyfile \
  -v caddy_data:/data \
  caddy:2
```

## 数据持久化

PG 数据在 `pg_data` named volume 里，重启/重建容器不会丢数据。