# AI 对话"全量输出"事故报告

> 2026-08-29 排查实录。从"用户反馈 AI 聊天每次都全量蹦出"到"修复完成",跨 6 轮迭代,涉及 nginx 配置、前端代码、dist 推送三个层面。
>
> 写这份文档的目的是:**给将来某天又遇到同样现象的同事/自己一份端到端的排查清单**,不重蹈"浏览器看起来全量=协议问题"的误判。

## 一、现象

- 用户在 `/admin/ai` 发消息,assistant 回复**等 1-3 秒后一次性蹦出完整内容**
- 视觉上像"非流式",但代码里明明是 `fetch().body.getReader()` + `ReadableStream` 解析 SSE

## 二、第一直觉(错的)

直觉反应:**前端代码问题**——split 错、buffer 错、Vue 反应式没触发。改了几次没生效。

**真实原因**:直觉跳过了**字节层验证**。在改前端之前,必须先确认**字节层是真流的**。

## 三、端到端字节层排查清单(必走)

按下列顺序,**每一步都必须独立验证**,不能跳:

| 步 | 工具 | 验证什么 | 关键命令 |
|---|---|---|---|
| 1 | 后端日志 | 后端 flush 是否频繁 | `tail /www/wwwlogs/go/blog_server.log` 看每次 delta 是否一条 `UPDATE ai_messages SET content = content \|\| '...'` |
| 2 | tcpdump lo | backend→nginx worker 字节流 | `tcpdump -ni lo -tttt "tcp port 8080"` 解 pcap 看 delta 间隔 |
| 3 | tcpdump eth0 | **nginx→浏览器字节流**(关键) | `tcpdump -ni eth0 -tttt "tcp port 80"` 解 pcap 看 delta 间隔 |
| 4 | Node.js fetch | 模拟浏览器 fetch + getReader | 写 30 行 Node 脚本用 `fetch + body.getReader()` 走一遍,记录每 chunk 到达时间 |

**判定标准**:任何一步 delta 间隔 p50 > 200ms 或 max > 2s,即该层在缓冲。

## 四、这次的三层问题

### 问题 1:nginx gzip 缓冲吞 SSE

**现象**(eth0 抓包):

```
=== 223.104.88.241:61479 -> 192.168.200.164:80  pkts=6 bytes=4326 ===
 0    0.000s   790   ← 立即出第一个大包
    1    0.104s   760
    2    3.154s   821   ← 卡了 3 秒!
    3    4.164s   739
    4    8.369s   645   ← 卡了 4 秒!
    5    8.495s   571
  gaps: p50=1010ms p90=4205ms
```

**根因**:nginx 顶层 `gzip on; gzip_min_length 1k; gzip_buffers 4 16k;`——gzip 会缓冲到 16k 才 flush,SSE 每个 data帧只有 30-50 字节,被攒成几秒一次的"大块"。

**修复**(`/www/server/panel/vhost/nginx/101.126.22.219.conf`):

```nginx
location ^~ /api/v1/ai/ {
    gzip off;                          # ← 关键:对 AI 路径关 gzip
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Connection "";    # ← 配套:关掉默认 Connection close
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 300s;
    proxy_send_timeout 600s;
}
```

注意:`^~` 是"非正则最长前缀优先",会被同级的 `location ^~ /` 截胡——必须放在 `^~ /` 之前,且路径字符串**严格长于** `/`。

**改完验证**(eth0 抓包):

```
=== nginx -> 浏览器  pkts=27 ===
  0    0.000s   243   data: {...}
  1    0.001s    32
  2    0.001s    43
  ...
 25    0.377s    19   data: [DONE]
  gaps: p50=0.7ms p90=67.3ms max=158.4ms
```

✅ 字节层立刻变流。

### 问题 2:Vue 反应式 setter 合并渲染

**现象**:字节层已是真流(p50 1ms),但浏览器 DOM 仍是"等几秒一次性出现"。

**根因**:`aiMsg.content += obj.delta` 这种字符串拼接,在 Vue 3 里:

- `messages = ref<Msg[]>([])` 创建的数组里 `aiMsg` 是**普通对象**,不是 reactive proxy
- `aiMsg.content +=` 不会触发响应式 setter
- Vue 没有 patch DOM,气泡 textContent 一直保持空
- 流式结束后 `aiMsg.pending = false` 触发 v-if 重渲染,此时 `aiMsg.content` 已经是完整字符串
- 一次 patch 显示完整内容——视觉上"全量"

**修复**(`web/src/views/admin/AICenter.vue`):**绕过 Vue,直接操作 DOM 文本节点**。

```typescript
// send 函数里,push aiMsg 之后
let bubbleEl: HTMLElement | null = null
await nextTick()
bubbleEl = document.querySelector<HTMLElement>('.msg.assistant:last-child .bubble')

// 解析 SSE 时
} else if (obj.delta) {
  if (bubbleEl && bubbleEl.firstChild) {
    bubbleEl.firstChild.nodeValue += obj.delta  // 直接改文本节点,不走 Vue
  }
  // 注意:不要写 aiMsg.content +=,会触发 Vue 重渲染,firstChild 引用失效
}
```

**为什么能 work**:Vue 模板 `{{ m.content }}` 渲染时会在 bubble 内生成第一个文本节点(初始空),cursor span 在它后面。**只改 firstChild.nodeValue,不碰 cursor span**,Vue 的 v-if 重渲染(cursor 消失)不影响 firstChild 引用。

**验证**(Node.js 浏览器行为模拟器,30 行):

```js
// 完整代码见 web/browser_sim.js,核心是:
const reader = resp.body.getReader()
const decoder = new TextDecoder()
let buffer = ''
while (...) {
  buffer += decoder.decode(value, { stream: true })
  const frames = buffer.split('\n\n')
  buffer = frames.pop() || ''
  for (const frame of frames) {
    const obj = JSON.parse(frame.match(/^data:\s*(.*)$/)[1])
    if (obj.delta) bubbleEl.firstChild.nodeValue += obj.delta
  }
}
```

实测结果:

```
[sim] 22 snapshots
[sim] +0.0ms len=0
[sim] +1103.6ms len=26    ← 第 1 个 delta
[sim] +1103.6ms len=48
[sim] +1142.8ms len=81
...
[sim] DOM-update gaps: p50=5.48ms p90=85.67ms
```

✅ Node 模拟器里 DOM 逐字更新,跟后端节奏一致。

### 问题 3:增量 scp 漏 chunk 导致浏览器白屏

**现象**:前端改完、`Ctrl+Shift+R` 后浏览器白屏。

**根因**:Vite 用 content-based hash 命名 chunk,**每次 build 几乎所有页面 chunk 都会变 hash**(因为依赖树牵动)。我前面用 `scp dist/assets/AICenter-*.js ...` 这种挑文件 scp,漏推了:

- `Login-DZj8v2NS.js`
- `api-BXw0_RPo.js`
- `ai-Cgh8m-7a.js`
- `ArticleList-CBjQlUD7.js`
- `ArticleEditor-Dfmdt6cd.js`
- ...等几十个新 hash

**这些 chunk 已被新 router 引用,但线上 dist 没文件**,浏览器请求时 404,整个 SPA 启动失败。

**修复**:**永远用 `scp -r dist/.` 全量覆盖**,不要挑文件:

```bash
scp -i <key> -r web/dist/. root@server:/www/wwwroot/blog-ui/dist/
scp -i <key> -r web/dist/. root@server:/www/wwwroot/blog-server/web/
```

(后端 `web/` 是 Go 直接 serve 的目录,见 `server/internal/router/router.go:20-22` 的 `r.Static("/assets", "./web/assets")`)

**校验命令**:

```bash
# 验证线上 index.html 引用的所有 chunk 都能 200
for f in $(curl -s http://server/ | grep -oE 'assets/[A-Za-z0-9_.-]+\.(js|css)' | sort -u); do
  curl -sSI http://server/$f | head -1
done
```

## 五、教训 / 排查清单(将来用)

下次看到"AI 对话全量输出",按下面顺序查,**每一步确认 OK 再走下一步**:

```
[ ] 1. 后端 SSE flush 是否频繁?
    tail /www/wwwlogs/go/blog_server.log,数 UPDATE ai_messages 频率

[ ] 2. tcpdump lo (backend→nginx) p50 < 100ms?
    tcpdump -ni lo -tttt "tcp port 8080" → 解 pcap

[ ] 3. tcpdump eth0 (nginx→浏览器) p50 < 100ms?
    tcpdump -ni eth0 -tttt "tcp port 80" → 解 pcap
    ↑ 这一步最容易栽:gzip / proxy_buffering / Connection: close 都是隐形杀手

[ ] 4. Node.js fetch 模拟器 p50 < 100ms?
    写 fetch + getReader 脚本,跟 AICenter 同样的解析代码

[ ] 5. 浏览器 DOM 仍"全量"?
    → Vue 反应式问题,改用 firstChild.nodeValue 直接 DOM 拼接

[ ] 6. 浏览器白屏 / 404?
    → 增量 scp 漏 chunk,必须 scp -r 全量覆盖

[ ] 7. 浏览器 console 报错 / Network 4xx?
    → 看具体哪个 chunk / API 缺失,补推
```

## 六、关键文件清单

| 文件 | 作用 |
|---|---|
| `server/internal/handler/ai.go` | 后端 SSE handler,`streamChatWithPersist` 每 delta `flusher.Flush()` |
| `server/internal/service/ai.go:143` | `AppendDelta` 用 PG `\|\|` 字符串拼接,每 delta 一条 UPDATE |
| `/www/server/panel/vhost/nginx/101.126.22.219.conf` | **线上** nginx 站点配置(宝塔),不是仓库里的 `web/nginx.conf` |
| `web/src/views/admin/AICenter.vue` | 前端 send 函数,`bubbleEl.firstChild.nodeValue += delta` |
| `web/browser_sim.js` | Node.js 浏览器行为模拟器(临时脚本,验证完可删) |
| `/www/wwwroot/blog-ui/dist/` | nginx serve 的前端产物 |
| `/www/wwwroot/blog-server/web/` | Go 后端 serve 的前端产物(同上内容) |
| `/www/wwwlogs/go/blog_server.log` | 后端 GORM 日志,每条 UPDATE 一行 |

## 七、为什么"换协议"不能解决

考虑过换 WebSocket / EventSource,结论:**底层协议换了,但响应式 + gzip 缓冲问题还在**——Vue 拿到 ws.onmessage 的 delta 后,还是要 `msg.content +=`,还是会触发同样问题。

**SSE 是对的协议**,问题在它**两侧的渲染/缓冲**。

## 八、相关规约

- 后端 SSE 实现细节见 `server/internal/handler/ai.go`,不要在 handler 里改 `flusher.Flush()` 的位置
- 前端 chunk 推送规约见本仓库 README(若需新增,可补一节"前端 dist 部署须 scp -r 全量")
- 宝塔部署细节见 `doc/deploy-bt.md`