<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { aiApi, type AIConversation, type AIMessage } from '../../api/ai'
import { useUserStore } from '../../stores/user'

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
  await ElMessageBox.confirm('确定删除该会话?会话内所有消息也会删除。', '提示', {
    type: 'warning',
  })
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
  // 拿到 assistant 气泡的 DOM 节点(绕过 Vue 反应式,直接写 textContent)
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
            // 绕过 Vue:直接操作 bubble 内的第一个文本节点,避免覆盖 cursor span
            // 不写 aiMsg.content,避免触发 Vue 重渲染覆盖 firstChild 引用
            if (bubbleEl && bubbleEl.firstChild) {
              bubbleEl.firstChild.nodeValue += obj.delta
            }
          }
        } catch {}
      }
      scrollToBottom()
    }
    aiMsg.pending = false
    // 流式结束后刷新会话列表（标题可能被自动设置）
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
  <div class="ai-page">
    <aside class="sidebar">
      <div class="sidebar-header">
        <span class="title">AI 会话</span>
        <el-button size="small" type="primary" @click="newConversation">+ 新对话</el-button>
      </div>
      <div class="conv-list">
        <div
          v-for="c in conversations"
          :key="c.id"
          :class="['conv-item', { active: c.id === currentConvId }]"
          @click="selectConversation(c.id)"
        >
          <span class="conv-title">{{ c.title || '未命名会话' }}</span>
          <el-button
            link
            size="small"
            class="del-btn"
            @click="removeConversation(c.id, $event)"
          >删除</el-button>
        </div>
        <div v-if="!conversations.length" class="empty">还没有会话,点上方新建</div>
      </div>
    </aside>

    <section class="chat-pane">
      <div class="toolbar">
        <h2>AI 对话 (MiniMax-M3)</h2>
        <div>
          <el-button :disabled="!currentConvId" @click="openRename">重命名</el-button>
        </div>
      </div>

      <el-card class="chat-card">
        <div ref="scrollBox" class="messages">
          <div v-for="(m, i) in messages" :key="i" :class="['msg', m.role]">
            <div class="bubble">
              {{ m.content }}<span v-if="m.pending" class="cursor">▍</span>
            </div>
          </div>
          <div v-if="!messages.length" class="hint">开始对话吧</div>
        </div>

        <div class="composer">
          <el-input
            v-model="input"
            type="textarea"
            :rows="2"
            placeholder="输入消息,Enter 发送,Shift+Enter 换行"
            :disabled="sending || !currentConvId"
            @keydown.enter.exact.prevent="send"
          />
          <el-button
            type="primary"
            :loading="sending"
            :disabled="!currentConvId"
            style="margin-left: 8px"
            @click="send"
          >发送</el-button>
        </div>
      </el-card>
    </section>

    <el-dialog v-model="renameDialogVisible" title="重命名会话" width="400px">
      <el-input v-model="renameValue" maxlength="100" show-word-limit placeholder="新标题" />
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRename">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.ai-page {
  display: flex;
  gap: 16px;
  height: calc(100vh - 120px);
}
.sidebar {
  width: 260px;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 6px;
  padding: 12px;
  border: 1px solid #ebeef5;
}
.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.sidebar-header .title {
  font-weight: 600;
}
.conv-list { flex: 1; overflow-y: auto; }
.conv-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  margin-bottom: 4px;
}
.conv-item:hover { background: #f5f7fa; }
.conv-item.active { background: #ecf5ff; color: #409eff; }
.conv-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.del-btn { visibility: hidden; }
.conv-item:hover .del-btn { visibility: visible; }
.empty { color: #909399; padding: 8px; font-size: 13px; }

.chat-pane { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.toolbar h2 { margin: 0; }
.chat-card { display: flex; flex-direction: column; flex: 1; }
.messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
  margin-bottom: 12px;
}
.msg { display: flex; margin-bottom: 12px; }
.msg.user { justify-content: flex-end; }
.msg.assistant { justify-content: flex-start; }
.bubble {
  max-width: 75%;
  padding: 10px 14px;
  border-radius: 10px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}
.msg.user .bubble { background: #409eff; color: #fff; }
.msg.assistant .bubble { background: #fff; border: 1px solid #ebeef5; }
.cursor { display: inline-block; animation: blink 1s infinite; }
@keyframes blink { 50% { opacity: 0; } }
.hint { color: #909399; text-align: center; padding: 40px 0; }
.composer { display: flex; align-items: flex-start; }
</style>