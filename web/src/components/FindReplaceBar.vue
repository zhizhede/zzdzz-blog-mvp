<script setup lang="ts">
import { computed, ref } from 'vue'
import { countMatches } from '../utils/findReplace'

// 查找替换条: 查找词实时上抛(parent 渲染高亮 overlay), 点「全部替换」上抛事件由 parent 改写正文.
// 与快捷用语条同一套嵌入方式: 编辑器 CONTENT 字段内一行, 不遮不挡.

const props = defineProps<{ text: string }>()
const emit = defineEmits<{
  (e: 'update:query', q: string): void
  (e: 'replace-all', search: string, replace: string): void
  (e: 'close'): void
}>()

const search = ref('')
const replace = ref('')

const matches = computed(() => countMatches(props.text, search.value))

const onSearchInput = () => emit('update:query', search.value)
const replaceAll = () => {
  if (search.value) emit('replace-all', search.value, replace.value)
}
const close = () => {
  emit('update:query', '')
  emit('close')
}
</script>

<template>
  <div class="find-bar">
    <input
      v-model="search"
      class="input f-input"
      placeholder="查找"
      @input="onSearchInput"
      @keydown.enter="replaceAll"
    />
    <input
      v-model="replace"
      class="input f-input"
      placeholder="替换为"
      @keydown.enter="replaceAll"
    />
    <span class="mono f-count">
      {{ search ? (matches ? `匹配 ${matches} 处` : '无匹配') : '' }}
    </span>
    <button type="button" class="f-btn" :disabled="!matches" @click="replaceAll">全部替换</button>
    <button type="button" class="f-btn ghost" @click="close">收起</button>
  </div>
</template>

<style scoped>
.find-bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.f-input {
  flex: 1;
  min-width: 120px;
  padding: 5px 0;
  font-size: 15.5px;
}
.f-count {
  color: var(--accent);
  font-size: 13.75px;
  letter-spacing: 0.04em;
  white-space: nowrap;
}
.f-btn {
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 4px 12px;
  font-family: var(--font-mono);
  font-size: 13.75px;
  color: var(--ink-soft);
  cursor: pointer;
  white-space: nowrap;
}
.f-btn:hover:not(:disabled) { color: var(--accent); border-color: var(--accent); }
.f-btn:disabled { color: var(--ink-faint); cursor: not-allowed; }
.f-btn.ghost { border-style: dashed; color: var(--ink-mute); }
</style>
