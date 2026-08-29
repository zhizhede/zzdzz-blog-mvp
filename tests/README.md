# tests/

本目录约定两种用途:

## `tests/<feature>/` — 正式测试用例(入仓库)

按功能拆分的单元 / 集成测试,通过 PR review 进入仓库,跑 CI。

- `tests/ai-streaming/` — AI 对话相关(流式 SSE、对话持久化)
- `tests/auth-admin/` — 后台管理员鉴权
- `tests/api-common/` — 通用 API

命名 / 目录结构按后续 test framework 引入再约定。

## `tests/_tmp/` — 临时调试脚本(**不入仓库**,被 .gitignore 排除)

排查线上问题 / 一次性验证脚本放这里。命名建议带前缀,便于 .gitignore 命中:

- `inspect_*.py` — 读线上配置 / chunk 内容(如 `inspect_dist.py`)
- `probe_*.js` — 字节层探针 / 网络抓包等价(Node.js 模拟浏览器 fetch)
- `patch_*.py` — 通过 SSH 改线上配置 / 数据库的修复脚本
- 其他一次性脚本任意命名,放在 `_tmp/` 子目录即可

特点:
- **不进 git**:整个 `tests/_tmp/` 目录被 `.gitignore` 排除,排查完可随时 `rm` 整个目录
- **不污染 git status**:`git status --short` 看不到它们,不影响正式 commit 的判断
- **跨会话留存**:即便临时脚本用完没删,下次还能复用或参考
- **避免混淆**:正式测试 (`tests/<feature>/`) 和临时调试(`tests/_tmp/`) 物理隔离

## 命名反例

禁止:
- ❌ 把临时脚本放 `tests/` 根目录或 `tests/<feature>/` 内 — 会污染正式测试
- ❌ 把临时脚本放项目根目录 — `.gitignore` 默认不忽略,容易误 commit
- ❌ 给临时脚本起正式名(如 `stream_test.js`)— 难以一眼区分"正式 vs 临时"

正确做法:
- ✅ `tests/_tmp/inspect_dist.py`
- ✅ `tests/_tmp/sse_probe.js`
- ✅ `tests/_tmp/patch_nginx.py`

## 当前 `tests/_tmp/` 里的脚本

> 本节是写文档时的快照,**不入仓库**(目录被 ignore)。仅作为示例。

| 文件 | 用途 | 触发场景 |
|---|---|---|
| `inspect_dist.py` | ssh 上 cat dist chunk, 看 hash 是否一致 | build 后验证 dist 是否完整推送 |
| `inspect_remote.py` | ssh 上读 AICenter 编译产物, 看 split 是不是 \n\n | 排查 split bug |
| `find_aimsg.py` | ssh 上找 compiled chunk 里 aiMsg 创建位置 | 排查 reactive 是否生效 |
| `sse_probe.js` | Node.js 模拟浏览器 fetch, 记录 delta 时间戳 | 字节层验证 |
| `browser_sim.js` | Node.js + 自建 DOM stub, 模拟 firstChild.nodeValue += | 验证直接 DOM 拼接逐字更新 |
| `patch_nginx.py` | 通过 ssh 改宝塔 nginx 站点配置, 加 /api/v1/ai/ gzip off | 修复 nginx gzip 缓冲吞 SSE |
| `patch_nginx2.py` | 同上, 第二次补丁(加完整 Host/X-Real-IP 头) | 完善 nginx 站点配置 |