# zzdzz-blog

个人博客系统 MVP，自用。基于 **Go + Gin + Gorm + PostgreSQL + Vue3 + Element Plus**。

---

## 功能

- 登录鉴权（JWT，30 天过期）
- 文章 CRUD（支持分类、搜索、分页、阅读量）
- 文章分类 CRUD
- AI 对话（OpenAI 兼容协议，默认接入 MiniMax-M3，支持流式 SSE）
- 单 SPA：后台管理 + 公开博客前台同站

## 目录结构

```
zzdzz-blog/
├── server/                  # Go 后端
│   ├── config/              # 配置加载 (Viper + YAML)
│   ├── internal/
│   │   ├── database/        # Gorm 数据库初始化
│   │   ├── handler/         # HTTP handler
│   │   ├── middleware/      # (预留)中间件
│   │   ├── model/           # Gorm 模型
│   │   ├── router/          # 路由注册
│   │   └── service/         # 业务逻辑
│   ├── migrations/          # SQL 迁移脚本
│   ├── pkg/
│   │   ├── jwt/             # JWT 生成 / 解析
│   │   └── response/        # 统一响应封装
│   ├── config.yaml          # 后端配置（含 DB / JWT / AI 配置）
│   ├── go.mod
│   └── main.go
├── web/                     # Vue3 前端
│   ├── src/
│   │   ├── api/             # axios 封装 + API 模块
│   │   ├── layouts/         # AdminLayout / BlogLayout
│   │   ├── router/          # Vue Router
│   │   ├── stores/          # Pinia stores
│   │   ├── styles/          # 全局样式
│   │   └── views/
│   │       ├── admin/       # 后台：文章 / 分类 / AI 对话
│   │       ├── public/      # 前台：博客列表 / 详情
│   │       └── Login.vue
│   ├── vite.config.ts       # 含 /api → 后端代理
│   └── package.json
└── doc/                     # 文档 / 规约
```

## 技术栈

**后端**
- Go 1.23+
- Gin (HTTP)
- Gorm + Postgres 驱动
- Viper (配置)
- Zap (日志)
- golang-jwt/jwt v5
- bcrypt
- sashabaranov/go-openai (OpenAI 兼容客户端)

**前端**
- Vue 3 + TypeScript
- Vite
- Element Plus + Icons
- Pinia (状态)
- Vue Router 4
- Axios

**数据库**
- PostgreSQL 16+ (建议启用 pgvector 扩展以备未来 RAG 用)

## 快速开始

### 前置条件
- Go 1.23+
- Node.js 20+
- PostgreSQL 14+ (本地或 Docker)
- 一个 OpenAI 兼容协议的 LLM 提供方（可选，本期不用 AI 也行）

### 1. 启动数据库

```bash
docker run -d --name pg \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=123456 \
  -e POSTGRES_DB=zzdzz_blog \
  -p 5432:5432 \
  postgres:15
```

### 2. 执行迁移

```bash
docker exec -i pg psql -U postgres -d zzdzz_blog < server/migrations/0001_init.sql
```

默认账号：`admin` / `123456`（密码已 bcrypt 哈希入库）

### 3. 启动后端

```bash
cd server
go mod download
go build -o bin/server.exe .
./bin/server.exe
# 监听 :8080
```

按需修改 `server/config/config.yaml`：
- `database.*`：DB 连接
- `jwt.secret`：JWT 签名密钥（生产环境务必改）
- `ai.base_url` / `ai.api_key` / `ai.model`：LLM 配置；不需要 AI 时 `enabled: false`

### 4. 启动前端

```bash
cd web
npm install
npm run dev
# 访问 http://localhost:5173
```

Vite dev server 已经把 `/api/*` 反代到 `http://localhost:8080`。

### 5. 登录

打开浏览器访问 http://localhost:5173，登录页已预填 `admin / 123456`。

## API 速览

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | `/api/v1/ping` | - | 健康检查 |
| POST | `/api/v1/auth/login` | - | 登录拿 token |
| GET | `/api/v1/auth/me` | ✅ | 当前用户 |
| GET | `/api/v1/categories` | - | 分类列表 |
| POST/PUT/DELETE | `/api/v1/categories[/:id]` | ✅ | 分类管理 |
| GET | `/api/v1/articles` | - | 文章分页 (`page`/`size`/`category_id`/`q`) |
| GET | `/api/v1/articles/:id` | - | 文章详情（自动 +1 阅读量）|
| POST/PUT/DELETE | `/api/v1/articles[/:id]` | ✅ | 文章 CRUD |
| POST | `/api/v1/ai/chat` | ✅ | AI 对话（支持 `stream:true` 走 SSE）|

## 部署

详见后续补充（计划用 Docker Compose 一把梭）。

## License

MIT