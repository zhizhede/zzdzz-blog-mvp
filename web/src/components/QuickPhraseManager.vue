<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { quickPhraseApi, type QuickPhrase } from '../api'

// 快捷用语板管理(增/改/删), 纯 UI 块不自带外框:
// 个人资料页内嵌一张卡片使用; 编辑页经 QuickPhraseDialog 包一层弹出使用.
// 失败提示由 http 拦截器统一弹出, 这里只管 loading 态.

const MAX_LEN = 200
const MAX_COUNT = 100

const items = ref<QuickPhrase[]>([])
const loading = ref(false)
const newContent = ref('')
const adding = ref(false)
// 正在就地改名的短语 id 与草稿值; null 表示没有编辑中的行
const editingId = ref<number | null>(null)
const editDraft = ref('')
const saving = ref(false)

const fetchList = async () => {
  loading.value = true
  try {
    const res = await quickPhraseApi.list()
    items.value = res.data ?? []
  } finally {
    loading.value = false
  }
}
onMounted(fetchList)

const add = async () => {
  const content = newContent.value.trim()
  if (!content) {
    ElMessage.warning('请输入快捷用语内容')
    return
  }
  if (items.value.length >= MAX_COUNT) {
    ElMessage.warning(`最多 ${MAX_COUNT} 条`)
    return
  }
  adding.value = true
  try {
    await quickPhraseApi.create(content)
    newContent.value = ''
    await fetchList()
    ElMessage.success('已添加')
  } finally {
    adding.value = false
  }
}

const startEdit = (p: QuickPhrase) => {
  editingId.value = p.id
  editDraft.value = p.content
}
const cancelEdit = () => {
  editingId.value = null
  editDraft.value = ''
}
const saveEdit = async () => {
  if (editingId.value == null) return
  const content = editDraft.value.trim()
  if (!content) {
    ElMessage.warning('内容不能为空')
    return
  }
  saving.value = true
  try {
    await quickPhraseApi.update(editingId.value, content)
    cancelEdit()
    await fetchList()
    ElMessage.success('已更新')
  } finally {
    saving.value = false
  }
}

const remove = async (p: QuickPhrase) => {
  const brief = p.content.length > 20 ? p.content.slice(0, 20) + '…' : p.content
  if (!window.confirm(`删除快捷用语「${brief}」?`)) return
  await quickPhraseApi.remove(p.id)
  await fetchList()
  ElMessage.success('已删除')
}
</script>

<template>
  <div class="qp-manager">
    <div class="add-row">
      <input
        v-model="newContent"
        class="input add-input"
        :placeholder="`输入常用语句,回车添加(最多 ${MAX_LEN} 字)`"
        :maxlength="MAX_LEN"
        @keydown.enter="add"
      />
      <button class="primary-btn" :disabled="adding || !newContent.trim()" @click="add">
        {{ adding ? '…' : '添加' }}
      </button>
    </div>

    <p v-if="loading" class="hint mono">加载中…</p>
    <p v-else-if="!items.length" class="hint">
      还没有快捷用语。添加后在写笔记页点一下,即可插入到正文光标处。
    </p>

    <ul v-else class="qp-list">
      <li v-for="p in items" :key="p.id" class="qp-item">
        <template v-if="editingId === p.id">
          <input
            v-model="editDraft"
            class="input edit-input"
            :maxlength="MAX_LEN"
            @keydown.enter="saveEdit"
            @keydown.esc="cancelEdit"
          />
          <button class="mini-btn accent" :disabled="saving" @click="saveEdit">保存</button>
          <button class="mini-btn" @click="cancelEdit">取消</button>
        </template>
        <template v-else>
          <span class="qp-text" :title="p.content">{{ p.content }}</span>
          <span class="qp-ops">
            <button class="mini-btn" @click="startEdit(p)">改</button>
            <button class="mini-btn danger" @click="remove(p)">删</button>
          </span>
        </template>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.qp-manager { display: flex; flex-direction: column; gap: 14px; }
.add-row { display: flex; gap: 12px; align-items: flex-end; }
.add-input { flex: 1; }
.input {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 8px 0;
  font-family: var(--font-body);
  font-size: 16.25px;
  color: var(--ink);
  outline: none;
  box-sizing: border-box;
}
.input:focus { border-bottom-color: var(--ink); }
.hint { color: var(--ink-mute); font-size: 15px; margin: 0; }
.qp-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
}
.qp-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 0;
  border-bottom: 1px dashed var(--rule-soft);
}
.qp-item:last-child { border-bottom: 0; }
.qp-text {
  flex: 1;
  color: var(--ink-soft);
  font-size: 16.25px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.edit-input { flex: 1; }
.qp-ops { display: flex; gap: 8px; flex-shrink: 0; }
.mini-btn {
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 3px 10px;
  font-family: var(--font-mono);
  font-size: 13.75px;
  color: var(--ink-soft);
  cursor: pointer;
}
.mini-btn:hover { color: var(--ink); border-color: var(--ink-mute); }
.mini-btn.accent { border-color: var(--accent); color: var(--accent); }
.mini-btn.danger:hover { color: var(--danger); border-color: var(--danger); }
.mini-btn:disabled { color: var(--ink-faint); cursor: not-allowed; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 8px 16px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 15px;
  letter-spacing: 0.08em;
  cursor: pointer;
  flex-shrink: 0;
}
.primary-btn:hover { background: var(--accent); }
.primary-btn:disabled { background: var(--ink-faint); cursor: not-allowed; }
</style>
