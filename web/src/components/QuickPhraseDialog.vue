<script setup lang="ts">
import QuickPhraseManager from './QuickPhraseManager.vue'

// 编辑页「快捷用语」管理弹窗: 自绘 overlay(与重置密码弹窗同一套样式),
// 管理逻辑全部在 QuickPhraseManager, 这里只负责弹层.
defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'closed'): void
}>()

const close = () => {
  emit('update:modelValue', false)
  emit('closed')
}
</script>

<template>
  <div v-if="modelValue" class="overlay" @click.self="close">
    <div class="dialog">
      <p class="mono d-tag">QUICK PHRASES</p>
      <h2 class="display d-title">快捷用语板</h2>
      <QuickPhraseManager />
      <div class="d-row">
        <button class="primary-btn" @click="close">完成</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog {
  background: var(--bg);
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  padding: 28px;
  width: 560px;
  max-width: calc(100vw - 48px);
  max-height: 80vh;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.d-tag { color: var(--accent); margin: 0; }
.d-title { font-size: 30px; margin: 0; }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 10px 18px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 15px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
}
.primary-btn:hover { background: var(--accent); }
</style>
