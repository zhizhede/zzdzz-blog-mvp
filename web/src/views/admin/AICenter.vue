<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { aiApi, type AIConversation, type AIMessage } from '../../api/ai'
import { useUserStore } from '../../stores/user'
import IssueTag from '../../components/IssueTag.vue'

// 同一组件挂在 /admin/ai 与 /space/ai 两处, 角标前缀按所在区域显示
const route = useRoute()
const issuePrefix = route.path.startsWith('/space') ? 'SPACE' : 'ADMIN'

// 引用来源: 后端召回命中时, 作为首个 SSE 事件的 sources 字段推送
interface RecallSource {
  index: number
  article_id: number
  title: string
  heading?: string
  scope: string
  score: number
}

interface Msg {
  id?: number
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
  sources?: RecallSource[]
}

const userStore = useUserStore()
const router = useRouter()
const conversations = ref<AIConversation[]>([])
const currentConvId = ref<number | null>(null)
const messages = ref<Msg[]>([])
const input = ref('')
const sending = ref(false)
const scrollBox = ref<HTMLElement | null>(null)
const aiPage = ref<HTMLElement | null>(null)
const renameDialogVisible = ref(false)
const renameValue = ref('')

// 聊天区占满视口剩余高度: 输入区钉在底部, 只有消息列表内部滚动。
// 高度实测而非写死 calc 常数, 页头/布局内边距以后调整也不会漂移。
const BOTTOM_RESERVE = 64
const chatHeight = ref('560px')
const measureChat = () => {
  const el = aiPage.value
  if (!el) return
  const top = el.getBoundingClientRect().top
  chatHeight.value = `${Math.max(420, window.innerHeight - top - BOTTOM_RESERVE)}px`
}

// force=false 时仅在用户本来就在底部附近才吸附到底, 避免流式输出把上翻阅读的用户拽回去
const scrollToBottom = async (force = true) => {
  await nextTick()
  const el = scrollBox.value
  if (!el) return
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
  if (force || nearBottom) el.scrollTop = el.scrollHeight
}

// MiniMax 系模型会在回答里输出 <think>…</think> 思考文本, 拆出来单独展示,
// 不和正文混在一起。thinking=true 表示流式输出还没收到 </think>(思考进行中)。
interface ParsedMsg {
  think: string
  answer: string
  thinking: boolean
}
const parseThink = (raw: string): ParsedMsg => {
  const open = raw.indexOf('<think>')
  if (open === -1) return { think: '', answer: raw, thinking: false }
  const close = raw.indexOf('</think>')
  const think = (close === -1 ? raw.slice(open + 7) : raw.slice(open + 7, close)).trim()
  const answer = (raw.slice(0, open) + (close === -1 ? '' : raw.slice(close + 8))).trim()
  return { think, answer, thinking: close === -1 }
}
const parsedOf = (m: Msg): ParsedMsg => parseThink(m.content)

// 思考块折叠状态: 键为消息下标; 未手动点过时, 思考进行中默认展开, 结束后默认收起
const thinkManual = ref<Record<number, boolean>>({})
const thinkVisible = (i: number, p: ParsedMsg) => thinkManual.value[i] ?? p.thinking
const toggleThink = (i: number) => {
  thinkManual.value[i] = !thinkVisible(i, parsedOf(messages.value[i]))
}

const loadConversations = async () => {
  const resp = await aiApi.listConversations()
  conversations.value = resp.data || []
  if (!currentConvId.value && conversations.value.length) {
    await selectConversation(conversations.value[0].id)
  }
}

const selectConversation = async (id: number) => {
  if (sending.value) {
    ElMessage.warning('当前会话还在生成,请稍候')
    return
  }
  currentConvId.value = id
  const resp = await aiApi.listMessages(id)
  messages.value = (resp.data || []).map((m: AIMessage) => ({
    id: m.id,
    role: m.role as 'user' | 'assistant',
    content: m.content,
  }))
  await scrollToBottom()
}

const newConversation = async () => {
  if (sending.value) {
    ElMessage.warning('当前会话还在生成,请稍候')
    return
  }
  const resp = await aiApi.createConversation()
  conversations.value.unshift(resp.data)
  await selectConversation(resp.data.id)
}

const removeConversation = async (id: number, ev: Event) => {
  ev.stopPropagation()
  await ElMessageBox.confirm('确定删除该会话?会话内所有消息也会删除。', '提示', { type: 'warning' })
  await aiApi.deleteConversation(id)
  conversations.value = conversations.value.filter((c) => c.id !== id)
  if (currentConvId.value === id) {
    currentConvId.value = null
    messages.value = []
    if (conversations.value.length) {
      await selectConversation(conversations.value[0].id)
    }
  }
}

const openRename = () => {
  if (!currentConvId.value) return
  const c = conversations.value.find((x) => x.id === currentConvId.value)
  if (!c) return
  renameValue.value = c.title === '未命名会话' ? '' : c.title
  renameDialogVisible.value = true
}

const submitRename = async () => {
  if (!currentConvId.value || !renameValue.value.trim()) {
    ElMessage.warning('标题不能为空')
    return
  }
  await aiApi.renameConversation(currentConvId.value, renameValue.value.trim())
  renameDialogVisible.value = false
  await loadConversations()
  ElMessage.success('已重命名')
}

const send = async () => {
  const text = input.value.trim()
  if (!text || sending.value || !currentConvId.value) return
  const convId = currentConvId.value

  messages.value.push({ role: 'user', content: text })
  input.value = ''
  // reactive 而非普通对象: delta 累积进状态驱动渲染, 不做任何直接 DOM 补丁
  const aiMsg = reactive<Msg>({ role: 'assistant', content: '', pending: true })
  messages.value.push(aiMsg)
  await scrollToBottom()
  sending.value = true

  try {
    const resp = await fetch(`/api/v1/ai/conversations/${convId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${userStore.token}`,
      },
      body: JSON.stringify({ content: text }),
    })

    if (!resp.ok || !resp.body) {
      const err = await resp.text()
      ElMessage.error(`请求失败: ${err}`)
      aiMsg.content = `错误: ${err}`
      aiMsg.pending = false
      return
    }

    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n\n')
      buffer = lines.pop() || ''
      for (const line of lines) {
        const m = line.match(/^data:\s*(.*)$/)
        if (!m) continue
        const payload = m[1]
        if (payload === '[DONE]') continue
        try {
          const obj = JSON.parse(payload)
          if (obj.error) {
            aiMsg.content = `错误: ${obj.error}`
          } else if (obj.sources) {
            // 召回引用来源, 早于 delta 到达
            aiMsg.sources = obj.sources
          } else if (obj.delta) {
            aiMsg.content += obj.delta
          }
        } catch {}
      }
      scrollToBottom(false)
    }
    aiMsg.pending = false
    await loadConversations()
  } catch (e: any) {
    aiMsg.content = `错误: ${e.message || e}`
    aiMsg.pending = false
  } finally {
    sending.value = false
    scrollToBottom(false)
  }
}

onMounted(() => {
  loadConversations()
  measureChat()
  requestAnimationFrame(measureChat)
  window.addEventListener('resize', measureChat)
})
onUnmounted(() => window.removeEventListener('resize', measureChat))
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div class="head-row">
        <div class="head-left">
          <IssueTag :prefix="issuePrefix" text="AI" suffix="MiniMax-M3" />
          <h1 class="display title">AI 对话</h1>
        </div>
        <div class="head-tools">
          <button class="text-btn" :disabled="!currentConvId" @click="openRename">重命名</button>
        </div>
      </div>
    </div>

    <div ref="aiPage" class="ai-page" :style="{ '--chat-h': chatHeight }">
      <aside class="sidebar">
        <div class="sidebar-head">
          <p class="mono tag">SESSIONS · {{ conversations.length }}</p>
          <button class="primary-btn" @click="newConversation">
            <span class="mono">＋</span> 新对话
          </button>
        </div>
        <div class="conv-list">
          <div
            v-for="c in conversations"
            :key="c.id"
            :class="['conv-item', { active: c.id === currentConvId }]"
            @click="selectConversation(c.id)"
          >
            <span class="conv-title">{{ c.title || '未命名会话' }}</span>
            <button
              class="text-btn del-btn"
              @click="removeConversation(c.id, $event)"
            >删除</button>
          </div>
          <div v-if="!conversations.length" class="empty">还没有会话,点上方新建。</div>
        </div>
      </aside>

      <section class="chat-pane">
        <div ref="scrollBox" class="messages">
          <div v-for="(m, i) in messages" :key="i" :class="['msg', m.role]">
            <div class="bubble">
              <span class="role-tag mono">{{ m.role === 'user' ? 'YOU' : 'AI' }}</span>
              <template v-if="m.role === 'assistant'">
                <div v-if="parsedOf(m).think || parsedOf(m).thinking" class="think-box">
                  <button class="think-toggle mono" @click="toggleThink(i)">
                    <span class="think-arrow">{{ thinkVisible(i, parsedOf(m)) ? '▾' : '▸' }}</span>
                    {{ parsedOf(m).thinking ? '思考中…' : '思考过程' }}
                  </button>
                  <div v-show="thinkVisible(i, parsedOf(m))" class="think-content">
                    {{ parsedOf(m).think }}<span v-if="parsedOf(m).thinking" class="cursor">▍</span>
                  </div>
                </div>
                <span class="bubble-text">
                  {{ parsedOf(m).answer }}<span v-if="m.pending" class="cursor">▍</span>
                </span>
              </template>
              <span v-else class="bubble-text">{{ m.content }}</span>
              <div v-if="m.sources?.length" class="src-row">
                <span class="mono src-label">引用</span>
                <button
                  v-for="s in m.sources"
                  :key="s.index"
                  class="src-chip mono"
                  :title="`相似度 ${(s.score * 100).toFixed(0)}% · ${s.scope === 'private' ? '私人笔记' : '公开文章'}`"
                  @click="router.push(`/blog/a/${s.article_id}`)"
                >
                  <span class="src-idx">[{{ s.index }}]</span>
                  {{ s.title }}<template v-if="s.heading"> · {{ s.heading }}</template>
                </button>
              </div>
            </div>
          </div>
          <div v-if="!messages.length" class="hint">开始对话吧</div>
        </div>

        <div class="composer">
          <textarea
            v-model="input"
            class="composer-input"
            rows="2"
            placeholder="输入消息,Enter 发送,Shift+Enter 换行"
            :disabled="sending || !currentConvId"
            @keydown.enter.exact.prevent="send"
          />
          <button
            class="primary-btn"
            :disabled="sending || !currentConvId"
            @click="send"
          >
            <span v-if="sending" class="mono">…</span>
            <span v-else class="mono">→</span>
            发送
          </button>
        </div>
      </section>
    </div>

    <div v-if="renameDialogVisible" class="overlay" @click.self="renameDialogVisible = false">
      <div class="dialog">
        <p class="mono d-tag">RENAME SESSION</p>
        <h2 class="display d-title">重命名会话</h2>
        <label class="field">
          <span class="mono label">TITLE</span>
          <input v-model="renameValue" class="input" maxlength="100" placeholder="新标题" />
        </label>
        <div class="d-row">
          <button class="text-btn" @click="renameDialogVisible = false">取消</button>
          <button class="primary-btn" @click="submitRename">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 压缩本页垂直装饰, 把纵向空间让给消息阅读区 (全站字号统一在全局缩放) */
.page {
  padding: 16px 0 0;
}
.page-head { display: flex; flex-direction: column; gap: 8px; margin-bottom: 0; padding-top: 12px; }
.head-row { display: flex; justify-content: space-between; align-items: center; }
.head-left { display: flex; align-items: center; gap: 14px; }
.title { font-size: 25px; line-height: 1; margin: 0; letter-spacing: -0.3px; }
.head-tools { display: flex; gap: 12px; }

.ai-page { display: flex; gap: 16px; height: var(--chat-h, 560px); }

.sidebar {
  width: 220px;
  display: flex;
  flex-direction: column;
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 16px;
  gap: 12px;
}
.sidebar-head { display: flex; justify-content: space-between; align-items: center; padding-bottom: 8px; border-bottom: 1px solid var(--rule-soft); }
.sidebar-head .tag { color: var(--ink-mute); margin: 0; font-size: 15px; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 8px 14px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 13.75px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.primary-btn:hover { background: var(--accent); }
.primary-btn:disabled { background: var(--ink-faint); cursor: not-allowed; }
.text-btn {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 4px 0;
  font-family: var(--font-body);
  font-size: 16.25px;
  color: var(--ink);
  cursor: pointer;
}
.text-btn:hover { color: var(--accent); border-bottom-color: var(--accent); }
.text-btn:disabled { color: var(--ink-faint); cursor: not-allowed; border-bottom-color: transparent; }

.conv-list { flex: 1; min-height: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 2px; }
.conv-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  border-radius: var(--radius);
  cursor: pointer;
  gap: 6px;
}
.conv-item:hover { background: var(--bg-sunken); }
.conv-item.active { background: var(--ink); color: var(--ink-on-inverse); }
.conv-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 16.25px; }
.del-btn { visibility: hidden; font-size: 13.75px; }
.conv-item:hover .del-btn { visibility: visible; color: var(--danger); }
.conv-item.active .del-btn { color: var(--ink-on-inverse); border-color: var(--ink-on-inverse); }
.empty { color: var(--ink-mute); padding: 12px; font-size: 15px; text-align: center; }

.chat-pane { flex: 1; display: flex; flex-direction: column; min-width: 0; gap: 12px; }
.messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px 18px;
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.msg { display: flex; }
.msg.user { justify-content: flex-end; }
.msg.assistant { justify-content: flex-start; }
.bubble {
  max-width: 78%;
  padding: 12px 16px;
  border-radius: var(--radius);
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.7;
  font-size: 17.5px;
  border: 1px solid var(--rule-soft);
}
.msg.user .bubble { background: var(--ink); color: var(--ink-on-inverse); border-color: var(--ink); }
/* AI 回答是主要阅读对象, 放开宽度限制占满整行 */
.msg.assistant .bubble { background: var(--bg); color: var(--ink); max-width: 100%; }
.role-tag {
  display: inline-block;
  font-size: 12.5px;
  letter-spacing: 0.16em;
  color: var(--ink-mute);
  margin-right: 8px;
  text-transform: uppercase;
}
.msg.user .role-tag { color: var(--ink-on-inverse); opacity: 0.7; }
.cursor { display: inline-block; animation: blink 1s infinite; margin-left: 2px; }
@keyframes blink { 50% { opacity: 0; } }
/* AI 思考文本: 与正文分开展示, 折叠可查 */
.think-box {
  border: 1px dashed var(--rule);
  border-radius: var(--radius);
  background: var(--bg-sunken);
  padding: 8px 12px;
  margin-bottom: 10px;
}
.think-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 0;
  padding: 0;
  font-size: 13.75px;
  letter-spacing: 0.1em;
  color: var(--ink-mute);
  cursor: pointer;
}
.think-toggle:hover { color: var(--accent); }
.think-arrow { font-size: 12.5px; }
/* 思考文本展开时必须全文铺开 (滚动交给外层消息列表), 禁止 max-height + 内部滚动条 */
.think-content {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--rule);
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 16.25px;
  line-height: 1.7;
  color: var(--ink-mute);
}
.src-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px solid var(--rule-soft);
}
.src-label { font-size: 12.5px; letter-spacing: 0.16em; color: var(--ink-mute); text-transform: uppercase; }
.src-chip {
  background: var(--bg-sunken);
  border: 1px solid var(--rule-soft);
  border-radius: 999px;
  padding: 3px 10px;
  font-size: 13.75px;
  color: var(--ink);
  cursor: pointer;
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.src-chip:hover { border-color: var(--accent); color: var(--accent); }
.src-idx { color: var(--ink-mute); margin-right: 4px; }
.hint { color: var(--ink-mute); text-align: center; padding: 40px 0; margin: auto; font-style: italic; }
.composer { display: flex; gap: 8px; align-items: stretch; }
.composer-input {
  flex: 1;
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 12px;
  font-family: var(--font-body);
  font-size: 17.5px;
  color: var(--ink);
  outline: none;
  resize: vertical;
  min-height: 56px;
}
.composer-input:focus { border-color: var(--ink); }

.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg); border: 1px solid var(--rule); border-radius: var(--radius); padding: 28px; width: 420px; display: flex; flex-direction: column; gap: 16px; }
.d-tag { color: var(--accent); margin: 0; font-size: 15px; }
.d-title { font-size: 30px; margin: 0; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 13.75px; text-transform: uppercase; letter-spacing: 0.16em; }
.input { background: transparent; border: 0; border-bottom: 1px solid var(--rule-soft); padding: 8px 0; font-family: var(--font-body); font-size: 17.5px; color: var(--ink); outline: none; }
.input:focus { border-bottom-color: var(--ink); }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }

/* 窄屏放弃视口内定高, 退回常规文档流, 会话列表与消息区各自限高滚动 */
@media (max-width: 900px) {
  .ai-page { flex-direction: column; height: auto; }
  .sidebar { width: auto; }
  .conv-list { flex: none; max-height: 180px; }
  .messages { flex: none; max-height: 62vh; }
}
</style>