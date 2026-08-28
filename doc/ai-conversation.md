# AI 对话持久化规约

> 本文档解释 `ai_conversations` / `ai_messages` 的设计取舍与行为细节,给将来修改这块代码的自己看。

## 数据模型

- `ai_conversations(id, user_id, title, created_at, updated_at)`
  - `ON DELETE CASCADE`: 删 `users` 行会自动清掉其所有会话
  - 索引 `(user_id, updated_at DESC)`: 会话列表按"最近活跃"排序只走索引
- `ai_messages(id, conversation_id, role, content, created_at)`
  - `ON DELETE CASCADE`: 删会话自动清消息
  - `role` 在数据库侧 CHECK 约束为 `user|assistant|system`
  - 索引 `(conversation_id, created_at)`: 拉历史消息按时间排序走索引

## 标题自动生成

- 新建会话时默认标题为 `未命名会话`。
- 当会话内累计 **正好 2 条消息**（1 条 user + 1 条 assistant 完成）时,把首条 user 消息的前 **20 字**(Unicode rune)写入 `title`。
- 之后无论会话多少条消息,标题不再自动改;用户可在 UI 手动重命名。
- 重命名后,自动起标题逻辑不再生效(`title != '未命名会话'` 直接返回)。

## 上下文窗口

- 每次发消息前,从 DB 取**最近 20 条**消息作为 LLM 上下文,按时间升序。
- 超出 20 条的消息**直接丢弃**,不做摘要、不报错。
- 这是 MVP 阶段的简化: 单用户 + 单会话通常不会超过 20 轮。
- 后续如需"超长会话",可改为摘要模式: 每 20 条切一段,前一段压缩成 1 条 system 消息。

## 流式写入策略

- 用户消息: 发出时立即 `INSERT`。
- AI 回复: 流式开始时先 `INSERT` 一条 `content=''` 的占位行,记下 `msg_id`。
- 每个 SSE chunk 到达时:
  1. 数据库侧 `UPDATE ai_messages SET content = content || $delta WHERE id = $msg_id`(PG 文本拼接,原子)
  2. 同一个 chunk 也通过 SSE 推给前端
- 流式结束(`io.EOF`): 调 `FinalizeAssistant`,触发"标题自动生成判断" + `updated_at = NOW()`。
- **关键收益**: 用户在中途刷新页面,DB 里已经有部分内容;切回旧会话能恢复。

## 兼容性

- 旧版无状态 `POST /api/v1/ai/chat` **继续保留**,前端若未升级也不影响。
- 新版持久化接口在 `/api/v1/ai/conversations[/:id[/messages]]`。

## 已知边界

- 单用户自用场景,没考虑并发(同一会话同时开两个 tab 发消息)。
- 流式期间每个 chunk 一次 UPDATE,DB 写入频率高;数据量大时可加节流(比如 200ms 合并一次)。
- `finalize` 时如果 DB 出错,会通过 SSE 推一条 `error` 事件;但已经写入的 chunk 不会回滚。