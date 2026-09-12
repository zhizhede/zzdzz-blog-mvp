<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { authApi } from '../api'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const form = ref({ old_password: '', new_password: '', confirm: '' })
const saving = ref(false)

const close = () => emit('update:modelValue', false)

const handleSubmit = async () => {
  const { old_password, new_password, confirm } = form.value
  if (!old_password) {
    ElMessage.warning('请输入旧密码')
    return
  }
  if (new_password.length < 6) {
    ElMessage.warning('新密码至少 6 位')
    return
  }
  if (new_password !== confirm) {
    ElMessage.warning('两次新密码不一致')
    return
  }
  saving.value = true
  try {
    await authApi.changeOwnPassword(old_password, new_password)
    ElMessage.success('密码已更新')
    form.value = { old_password: '', new_password: '', confirm: '' }
    close()
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-if="modelValue" class="overlay" @click.self="close">
    <div class="dialog">
      <p class="mono d-tag">RESET PASSWORD</p>
      <h2 class="display d-title">重置密码</h2>
      <label class="field">
        <span class="mono label">OLD PASSWORD</span>
        <input v-model="form.old_password" type="password" class="input" />
      </label>
      <label class="field">
        <span class="mono label">NEW PASSWORD · 至少 6 位</span>
        <input v-model="form.new_password" type="password" class="input" />
      </label>
      <label class="field">
        <span class="mono label">CONFIRM</span>
        <input v-model="form.confirm" type="password" class="input" />
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
.d-title { font-size: 24px; margin: 0; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.input { background: transparent; border: 0; border-bottom: 1px solid var(--rule-soft); padding: 8px 0; font-family: var(--font-body); font-size: 14px; color: var(--ink); outline: none; }
.input:focus { border-bottom-color: var(--ink); }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 10px 18px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
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
  font-size: 13px;
  color: var(--ink);
  cursor: pointer;
}
.text-btn:hover { color: var(--accent); border-bottom-color: var(--accent); }
</style>
