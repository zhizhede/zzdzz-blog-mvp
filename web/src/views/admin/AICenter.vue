<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../../stores/user'

interface Msg { role: 'user' | 'assistant'; content: string; pending?: boolean }

const userStore = useUserStore()
const messages = ref<Msg[]>([
  { role: 'assistant', content: '你好,我是 MiniMax-M3,有什么想聊的?' },
])
const input = ref('')
const sending = ref(false)
const scrollBox = ref<HTMLElement | null>(null)

const scrollToBottom = async () => {
  await nextTick()
  if (scrollBox.value) scrollBox.value.scrollTop = scrollBox.value.scrollHeight
}

const send = async () => {
  const text = input.value.trim()
  if (!text || sending.value) return
  messages.value.push({ role: 'user', content: text })
  input.value = ''
  const aiMsg: Msg = { role: 'assistant', content: '', pending: true }
  messages.value.push(aiMsg)
  await scrollToBottom()
  sending.value = true

  try {
    const payload = {
      messages: messages.value
        .filter((m) => !m.pending)
        .map((m) => ({ role: m.role, content: m.content })),
      stream: true,
    }

    const resp = await fetch('/api/v1/ai/chat', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${userStore.token}`,
      },
      body: JSON.stringify(payload),
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
          } else if (obj.delta) {
            aiMsg.content += obj.delta
          }
        } catch {}
      }
      scrollToBottom()
    }
    aiMsg.pending = false
  } catch (e: any) {
    aiMsg.content = `错误: ${e.message || e}`
    aiMsg.pending = false
  } finally {
    sending.value = false
    scrollToBottom()
  }
}

const clearChat = () => {
  messages.value = [{ role: 'assistant', content: '已清空,开始新对话。' }]
}
</script>

<template>
  <div class="page-container chat-container">
    <div class="toolbar">
      <h2>AI 对话 (MiniMax-M3)</h2>
      <el-button @click="clearChat">清空</el-button>
    </div>

    <el-card class="chat-card">
      <div ref="scrollBox" class="messages">
        <div v-for="(m, i) in messages" :key="i" :class="['msg', m.role]">
          <div class="bubble">{{ m.content }}<span v-if="m.pending" class="cursor">▍</span></div>
        </div>
      </div>

      <div class="composer">
        <el-input
          v-model="input"
          type="textarea"
          :rows="2"
          placeholder="输入消息,Enter 发送,Shift+Enter 换行"
          :disabled="sending"
          @keydown.enter.exact.prevent="send"
        />
        <el-button type="primary" :loading="sending" @click="send" style="margin-left: 8px">
          发送
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.chat-container { max-width: 800px; }
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.toolbar h2 { margin: 0; }
.chat-card { display: flex; flex-direction: column; height: calc(100vh - 180px); }
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
.composer { display: flex; align-items: flex-start; }
</style>