<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { authApi } from '../api'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const password = ref('')
const hint = ref('')
const currentHint = ref('')
const saving = ref(false)

const close = () => emit('update:modelValue', false)

// 每次打开时拉当前提示, 方便对照修改
watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    password.value = ''
    hint.value = ''
    currentHint.value = ''
    try {
      const res = await authApi.me()
      currentHint.value = res.data.password_hint || ''
    } catch {
      // 拦截器已提示
    }
  },
  { immediate: true }
)

const handleSubmit = async () => {
  if (!password.value) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (hint.value.length > 255) {
    ElMessage.warning('提示最多 255 字')
    return
  }
  saving.value = true
  try {
    await authApi.changeOwnPasswordHint(password.value, hint.value.trim())
    ElMessage.success('密码提示已更新')
    close()
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-if="modelValue" class="overlay" @click.self="close">
    <div class="dialog">
      <p class="mono d-tag">CHANGE HINT</p>
      <h2 class="display d-title">更改密码提示</h2>
      <label class="field">
        <span class="mono label">CURRENT PASSWORD</span>
        <input v-model="password" type="password" class="input" />
      </label>
      <label class="field">
        <span class="mono label">NEW HINT · 选填, 忘记密码时可公开查看</span>
        <input
          v-model="hint"
          class="input"
          :placeholder="currentHint ? `当前提示: ${currentHint}` : '未设置提示'"
          maxlength="255"
        />
      </label>
      <div class="d-row">
        <button class="text-btn" @click="close">取消</button>
        <button class="primary-btn" :disabled="saving" @click="handleSubmit">
          <span v-if="saving" class="mono">…</span>
          <span v-else>保存</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg); border: 1px solid var(--rule); border-radius: var(--radius); padding: 28px; width: 420px; display: flex; flex-direction: column; gap: 16px; }
.d-tag { color: var(--accent); margin: 0; }
.d-title { font-size: 30px; margin: 0; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 13.75px; text-transform: uppercase; letter-spacing: 0.16em; }
.input { background: transparent; border: 0; border-bottom: 1px solid var(--rule-soft); padding: 8px 0; font-family: var(--font-body); font-size: 17.5px; color: var(--ink); outline: none; }
.input:focus { border-bottom-color: var(--ink); }
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
</style>
