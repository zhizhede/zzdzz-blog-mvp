<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { aiApi, type AIConversation, type AIMessage } from '../../api/ai'
import { useUserStore } from '../../stores/user'
import IssueTag from '../../components/IssueTag.vue'

interface Msg {
  id?: number
  role: 'user' | 'assistant'
  content: string
  pending?: boolean
}

const userStore = useUserStore()
const conversations = ref<AIConversation[]>([])
const currentConvId = ref<number | null>(null)
const messages = ref<Msg[]>([])
const input = ref('')
const sending = ref(false)
const scrollBox = ref<HTMLElement | null>(null)
const renameDialogVisible = ref(false)
const renameValue = ref('')

const scrollToBottom = async () => {
  await nextTick()
  if (scrollBox.value) scrollBox.value.scrollTop = scrollBox.value.scrollHeight
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
  const aiMsg: Msg = { role: 'assistant', content: '', pending: true }
  messages.value.push(aiMsg)
  await scrollToBottom()
  let bubbleEl: HTMLElement | null = null
  await nextTick()
  bubbleEl = document.querySelector<HTMLElement>('.msg.assistant:last-child .bubble')
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
            if (bubbleEl && bubbleEl.firstChild) {
              bubbleEl.firstChild.nodeValue = aiMsg.content
            }
          } else if (obj.delta) {
            if (bubbleEl && bubbleEl.firstChild) {
              bubbleEl.firstChild.nodeValue += obj.delta
            }
          }
        } catch {}
      }
      scrollToBottom()
    }
    aiMsg.pending = false
    await loadConversations()
  } catch (e: any) {
    aiMsg.content = `错误: ${e.message || e}`
    aiMsg.pending = false
  } finally {
    sending.value = false
    scrollToBottom()
  }
}

onMounted(loadConversations)
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="ADMIN" text="AI" suffix="MiniMax-M3" />
      <div class="head-row">
        <h1 class="display title">AI 对话</h1>
        <div class="head-tools">
          <button class="text-btn" :disabled="!currentConvId" @click="openRename">重命名</button>
        </div>
      </div>
    </div>

    <div class="ai-page">
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
              <span class="bubble-text">
                {{ m.content }}<span v-if="m.pending" class="cursor">▍</span>
              </span>
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
.page { display: flex; flex-direction: column; gap: 16px; padding-bottom: 32px; }
.page-head { display: flex; flex-direction: column; gap: 8px; }
.head-row { display: flex; justify-content: space-between; align-items: baseline; }
.title { font-size: 32px; line-height: 1; margin: 0; letter-spacing: -0.6px; }
.head-tools { display: flex; gap: 12px; }

.ai-page { display: flex; gap: 16px; min-height: calc(100vh - 200px); }

.sidebar {
  width: 260px;
  display: flex;
  flex-direction: column;
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 16px;
  gap: 12px;
}
.sidebar-head { display: flex; justify-content: space-between; align-items: center; padding-bottom: 8px; border-bottom: 1px solid var(--rule-soft); }
.sidebar-head .tag { color: var(--ink-mute); margin: 0; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 8px 14px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 11px;
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
  font-size: 13px;
  color: var(--ink);
  cursor: pointer;
}
.text-btn:hover { color: var(--accent); border-bottom-color: var(--accent); }
.text-btn:disabled { color: var(--ink-faint); cursor: not-allowed; border-bottom-color: transparent; }

.conv-list { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 2px; }
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
.conv-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; }
.del-btn { visibility: hidden; font-size: 11px; }
.conv-item:hover .del-btn { visibility: visible; color: var(--danger); }
.conv-item.active .del-btn { color: var(--ink-on-inverse); border-color: var(--ink-on-inverse); }
.empty { color: var(--ink-mute); padding: 12px; font-size: 12px; text-align: center; }

.chat-pane { flex: 1; display: flex; flex-direction: column; min-width: 0; gap: 12px; }
.messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  display: flex;
  flex-direction: column;
  gap: 14px;
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
  font-size: 14px;
  border: 1px solid var(--rule-soft);
}
.msg.user .bubble { background: var(--ink); color: var(--ink-on-inverse); border-color: var(--ink); }
.msg.assistant .bubble { background: var(--bg); color: var(--ink); }
.role-tag {
  display: inline-block;
  font-size: 10px;
  letter-spacing: 0.16em;
  color: var(--ink-mute);
  margin-right: 8px;
  text-transform: uppercase;
}
.msg.user .role-tag { color: var(--ink-on-inverse); opacity: 0.7; }
.cursor { display: inline-block; animation: blink 1s infinite; margin-left: 2px; }
@keyframes blink { 50% { opacity: 0; } }
.hint { color: var(--ink-mute); text-align: center; padding: 60px 0; font-style: italic; }
.composer { display: flex; gap: 8px; align-items: stretch; }
.composer-input {
  flex: 1;
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 12px;
  font-family: var(--font-body);
  font-size: 14px;
  color: var(--ink);
  outline: none;
  resize: vertical;
  min-height: 56px;
}
.composer-input:focus { border-color: var(--ink); }

.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg); border: 1px solid var(--rule); border-radius: var(--radius); padding: 28px; width: 420px; display: flex; flex-direction: column; gap: 16px; }
.d-tag { color: var(--accent); margin: 0; }
.d-title { font-size: 24px; margin: 0; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.input { background: transparent; border: 0; border-bottom: 1px solid var(--rule-soft); padding: 8px 0; font-family: var(--font-body); font-size: 14px; color: var(--ink); outline: none; }
.input:focus { border-bottom-color: var(--ink); }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }
</style>