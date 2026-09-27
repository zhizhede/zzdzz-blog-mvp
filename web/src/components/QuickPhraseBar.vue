<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { quickPhraseApi, type QuickPhrase } from '../api'
import QuickPhraseDialog from './QuickPhraseDialog.vue'

// 快捷用语板(编辑器内嵌条): 点击短语 → emit('insert', 内容) 由父级插入光标处;
// 「管理」打开弹窗增删改, 关闭后重拉列表保持板面最新.
// 点击短语会先 blur 正文框, selectionStart/End 仍保留, 父级插入后自行 focus 回来.

const emit = defineEmits<{ (e: 'insert', text: string): void }>()

const items = ref<QuickPhrase[]>([])
const loaded = ref(false)
const dialogOpen = ref(false)

const fetchList = async () => {
  try {
    const res = await quickPhraseApi.list()
    items.value = res.data ?? []
    loaded.value = true
  } catch {
    // 失败提示由 http 拦截器统一弹出; 板面保持隐藏, 不挡写作
  }
}
onMounted(fetchList)

// 管理弹窗关闭后刷新; 失败时静默(拦截器已提示)
const onDialogClosed = () => {
  fetchList()
}
</script>

<template>
  <div v-if="loaded" class="qp-bar">
    <span class="qp-label mono">快捷用语</span>
    <button
      v-for="p in items"
      :key="p.id"
      type="button"
      class="qp-chip"
      :title="`点击插入到光标处: ${p.content}`"
      @click="emit('insert', p.content)"
    >{{ p.content }}</button>
    <span v-if="!items.length" class="qp-empty">暂无,点右侧管理添加</span>
    <button type="button" class="qp-chip manage" title="管理快捷用语" @click="dialogOpen = true">
      ＋ 管理
    </button>
    <QuickPhraseDialog v-model="dialogOpen" @closed="onDialogClosed" />
  </div>
</template>

<style scoped>
.qp-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  max-height: 96px;
  overflow-y: auto;
}
.qp-label {
  color: var(--accent);
  font-size: 12.5px;
  letter-spacing: 0.12em;
  flex-shrink: 0;
  margin-right: 2px;
}
.qp-chip {
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: 999px;
  padding: 3px 10px;
  font-family: var(--font-body);
  font-size: 13.75px;
  color: var(--ink-soft);
  cursor: pointer;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: all var(--transition);
}
.qp-chip:hover { color: var(--accent); border-color: var(--accent); }
.qp-chip.manage {
  font-family: var(--font-mono);
  color: var(--ink-mute);
  border-style: dashed;
}
.qp-chip.manage:hover { color: var(--ink); border-color: var(--ink-mute); }
.qp-empty { color: var(--ink-mute); font-size: 13.75px; }
</style>
