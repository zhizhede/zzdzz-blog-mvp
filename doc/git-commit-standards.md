# Git 提交规约

> **拆分自 `lisay-govel-generator开发规约.md` §5（2026-08-27 拆分）**
>
> 适用范围与主规约一致：P2 及之后所有新代码必须遵循；P1 已存在代码沿用历史命名，不强制重写。
>
> **跨文件引用约定**：本文档章节编号独立（§1 ~ §7）；凡引用主规约章节（如 §3.5.6 迁移路径、§3.5.2 字段表），均指 `lisay-govel-generator开发规约.md` 对应章节。

---

## §1. Conventional Commits（约定式提交）

采用 [Conventional Commits](https://www.conventionalcommits.org/)（Angular 规范），行业最广泛使用的事实标准。所有 commit 必须同时满足 §1 格式规范 + §2 原子性规范，缺一不可。

### §1.1 标准格式模板

```
<type>(<scope>): <subject>

<空行>

<body>

<空行>

<footer>
```

### §1.2 type 变更类型（必填，固定取值）

| 值 | 语义 | 示例 |
|---|---|---|
| `feat` | 新功能（feature） | 新增 skill 热 reload |
| `fix` | 修复 bug | 修复 private 模型越权 |
| `docs` | 仅修改文档 / 注释 | 补充规约 / README |
| `style` | 代码格式调整（空格、分号、换行，不改动业务逻辑） | import 排序 |
| `refactor` | 代码重构（无新功能、无 bug 修复） | 抽公共基类 |
| `perf` | 性能优化 | 缓存命中率提升 |
| `test` | 新增 / 修改单元测试、接口测试 | 补单元测试 |
| `chore` | 构建脚本、依赖、工具配置修改 | `pom.xml` / `docker` / `.gitignore` |
| `build` | 构建系统 / 外部依赖变更（与 `chore` 类似，常合并使用） | 升级 Spring Boot 版本 |
| `ci` | CI 配置文件 / 脚本变更 | GitHub Actions / Jenkinsfile |
| `revert` | 回滚之前的 commit | `revert: feat(skill): ...` |

> **不发明新 type**：禁止 `feature` / `bug` / `update` / `misc` 等模糊取值，必须从上表选。

### §1.3 scope 作用域（选填，区分模块 / 表 / 功能）

| scope | 说明 |
|---|---|
| `ai` | AI 框架整体变更（跨 ai-skill / ai-llm / ai-rag） |
| `ai-skill` | Skill 模块 |
| `ai-llm` | LLM 模块（model-router / provider / model config） |
| `ai-model` | AI 模型配置（`ai_provider_config` / `ai_model_config` 表） |
| `ai-rag` | RAG / 知识库 |
| `auth-admin` | 后台管理员域 |
| `auth-client` | 用户端域 |
| `common` | common 公共模块 |
| `sql` | DDL / DML 迁移脚本 |
| `pom` | Maven 多模块 / 依赖管理 |
| `docker` | 容器化 / 编排 |
| `gitignore` | Git 忽略规则 |
| `redis-cache` | Redis 缓存层 |
| `model-router` | LLM 路由层 |

示例：`feat(ai-model): 新增 scope_type、owner_user_id 隔离字段`

### §1.4 subject 简短描述（必填）

- **50 字符以内**
- **动词开头**（现在时）：`add` / `fix` / `optimize` / `refactor` / `remove` / `support` / `upgrade`
- **不加句号**
- **禁止长句、模糊描述**（不要写"改了点东西"）

### §1.5 body 详细描述（可选）

- 写清楚**改动原因**、**实现逻辑**、**上下游影响**
- 每条单独一行，**72 字符换行**
- 用 `-` 列表化更清晰
- 涉及数据库变更时，**必须**写明 DDL 文件路径 + 受影响表名

示例：

```
feat(auth-admin): 实现 AdminUser DO 切基类

- 11 个 auth 域 DO 统一 extends BaseAuditEntity
- 类名加 DO 后缀：AdminUser → AdminUserDO
- AuditMetaObjectHandler 从 UserContext.getUserId() 取 Long 类型
- 删本地 @TableLogic（基类接管）
- 跑 migrate-1-统一软删除规范.sql 重命名 created_at → create_time

Refs: 主规约 §3.5.6 迁移路径
```

### §1.6 footer 尾部备注（可选）

1. 关联需求 / 缺陷单号：`Closes #123` / `Fix #456` / `Refs #789`
2. 破坏性变更：`BREAKING CHANGE: xxx`，**必须**写明迁移路径
3. 共同作者：`Co-authored-by: Name <name@example.com>`

---

## §2. 原子性提交（Atomic Commits）

每个 commit **只承载一个独立的逻辑变更**，便于 code review、cherry-pick、`git revert`、`git bisect` 二分定位。

### §2.1 核心原则（5 条）

| # | 原则 | 解释 |
|---|---|---|
| 1 | **一个 commit = 一件事** | 一个新功能 / 一个 bug 修复 / 一次重构 / 一次依赖升级，不混搭 |
| 2 | **提交后项目可构建** | commit 完成后 `mvn compile` 必须通过，不留"半成品状态"进版本库 |
| 3 | **提交后测试可运行** | 涉及业务逻辑必须有对应测试通过；纯文档 / 注释 / 配置变更不强制 |
| 4 | **不混入无关改动** | 顺手做的格式化 / IDE 调整 / 顺手优化要么独立成 commit，要么拆 PR |
| 5 | **粒度适中，不要过碎** | 一个完整特性可拆 3~8 个 commit；过细（如每行一次 commit）反而增加 review 成本 |

### §2.2 拆分粒度指南

| 场景 | 推荐拆法 |
|---|---|
| 涉及 DO 重构 + Service 调用点 + Controller 接口变更 | **拆 3 个 commit**：① 新建 DO / 重构基类 ② 替换 Service 调用 ③ Controller 切换 |
| SQL 迁移 + Java 实体 + Handler + 业务代码 | **拆 4 个 commit**：① DDL 迁移脚本 ② Entity 切基类 ③ Handler 接入 ④ 业务上线 |
| 升级依赖 + 配套代码调整 | **拆 2 个 commit**：① 仅 `pom.xml` 版本号 ② 业务代码适配 |
| 修 bug 同时引入新 feature | **拆 2 个 commit**：① 先 fix ② 再 feat，禁止合一次 |
| PR review 中要求"顺便改的小问题" | **独立 commit**（fix / style 单独一条），不要塞进主 commit |
| 批量重命名（包路径迁移） | **按业务域拆 commit**，每个域一个，不要一次 commit 跨 5 个域 |

### §2.3 反面错误示例（禁止）

| 反面写法 | 为什么错 |
|---|---|
| `git commit -m "更新代码"` | 模糊无意义，无 type / scope |
| `git commit -m "新增模型隔离"` | 无 type 分类 |
| `git commit -m "fix: 修复路由缓存bug."` | 结尾带句号 |
| `git commit -m "feat: 改了很多东西"` | subject 模糊，无法回溯 |
| 一个 commit 同时改 DDL + Entity + Service + Controller + 文档 + 测试 | 原子性破环，无法 `git revert` / cherry-pick |
| 一个 commit 把 5 个不相关 bug fix 合并提交 | review 困难，bisect 定位困难 |
| 提交后 `mvn compile` 失败 / 测试红 | 半成品状态，污染 main 分支 |
| commit 信息与实际改动不符（如 `fix: ...` 实际是 `feat`） | 历史追溯失真 |

---

## §3. PR 自检清单（reviewer 必查）

每次 PR 涉及 commit 历史（squash / rebase / merge），**必须**核对：

- [ ] commit 信息符合 §1 格式（`<type>(<scope>): <subject>`）？
- [ ] `type` 取值在 §1.2 固定 11 取值内（不发明新词）？
- [ ] `scope` 取值符合 §1.3 模块表（不发明新 scope）？
- [ ] `subject` 50 字符内、动词开头、无句号？
- [ ] 涉及业务改动时 `body` 写清了"为什么 / 怎么改 / 影响面"？
- [ ] 涉及数据库变更时 `body` 写明 DDL 文件路径 + 受影响表名？
- [ ] `footer` 关联了 issue / 缺陷单号（`Closes #xxx` / `Fix #xxx`）？
- [ ] 涉及破坏性变更时写了 `BREAKING CHANGE: ...` + 迁移路径？
- [ ] **原子性**：一个 commit 只承载一个独立逻辑变更（详见 §2）？
- [ ] 提交后项目可编译（`mvn compile` / `pnpm build` 通过）？
- [ ] 没有把无关改动（顺手格式化 / 顺手优化）混入主 commit？
- [ ] `git log --oneline` 历史可读性强（不出现"update"、"fix bug"等模糊描述）？

**违反任一项的 PR 直接打回，要求修改 commit 信息或拆分 commit 后再 review。

---

## 
