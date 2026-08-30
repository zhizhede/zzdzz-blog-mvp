<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { authApi } from '../../api'
import { useUserStore } from '../../stores/user'
import IssueTag from '../../components/IssueTag.vue'

const userStore = useUserStore()

const passwordDialog = ref(false)
const passwordForm = ref({ old_password: '', new_password: '', confirm: '' })
const saving = ref(false)

const handleChangePassword = async () => {
  const { old_password, new_password, confirm } = passwordForm.value
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
    passwordDialog.value = false
    passwordForm.value = { old_password: '', new_password: '', confirm: '' }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="SPACE" text="PROFILE" suffix="ACCOUNT" />
      <h1 class="display title">个人资料</h1>
    </div>

    <div class="grid">
      <section class="card">
        <p class="mono card-tag">IDENTITY</p>
        <div class="identity">
          <span class="avatar" :class="userStore.isAdmin ? 'admin' : 'reader'">
            {{ userStore.username.charAt(0).toUpperCase() }}
          </span>
          <div class="who">
            <h2 class="display name">{{ userStore.username }}</h2>
            <p class="mono uid">#{{ userStore.userId }} · {{ userStore.isAdmin ? 'admin' : 'reader' }}</p>
          </div>
        </div>
        <div class="actions">
          <button class="primary-btn" @click="passwordDialog = true">
            <span class="mono">↻</span> 重置密码
          </button>
        </div>
      </section>

      <section class="card">
        <p class="mono card-tag">SECURITY</p>
        <ul class="tips">
          <li><span class="mono dot">●</span> 密码长度至少 6 位</li>
          <li><span class="mono dot">●</span> 重置后会自动退出旧会话</li>
          <li><span class="mono dot">●</span> 自己账号需输入当前密码</li>
        </ul>
      </section>
    </div>

    <div v-if="passwordDialog" class="overlay" @click.self="passwordDialog = false">
      <div class="dialog">
        <p class="mono d-tag">RESET PASSWORD</p>
        <h2 class="display d-title">重置密码</h2>
        <label class="field">
          <span class="mono label">OLD PASSWORD</span>
          <input v-model="passwordForm.old_password" type="password" class="input" />
        </label>
        <label class="field">
          <span class="mono label">NEW PASSWORD · 至少 6 位</span>
          <input v-model="passwordForm.new_password" type="password" class="input" />
        </label>
        <label class="field">
          <span class="mono label">CONFIRM</span>
          <input v-model="passwordForm.confirm" type="password" class="input" />
        </label>
        <div class="d-row">
          <button class="text-btn" @click="passwordDialog = false">取消</button>
          <button class="primary-btn" :disabled="saving" @click="handleChangePassword">
            <span v-if="saving" class="mono">…</span>
            <span v-else>保存</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 24px; padding-bottom: 64px; }
.page-head { display: flex; flex-direction: column; gap: 12px; }
.title { font-size: 36px; line-height: 1; margin: 0; letter-spacing: -0.8px; }

.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.card {
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.card-tag { color: var(--accent); margin: 0; }
.identity { display: flex; gap: 16px; align-items: center; }
.avatar {
  width: 56px; height: 56px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 600;
  border: 1px solid var(--rule-soft);
}
.avatar.admin { background: var(--accent); color: var(--accent-ink); border-color: var(--accent); }
.avatar.reader { background: var(--bg-sunken); color: var(--ink); }
.who { display: flex; flex-direction: column; gap: 4px; }
.name { font-size: 24px; margin: 0; }
.uid { font-size: 12px; color: var(--ink-mute); }
.actions { display: flex; gap: 12px; padding-top: 8px; border-top: 1px solid var(--rule-soft); }
.tips { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.tips li { display: flex; gap: 10px; align-items: baseline; color: var(--ink-soft); font-size: 14px; }
.tips .dot { color: var(--accent); }

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
  display: inline-flex;
  align-items: center;
  gap: 6px;
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

.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg); border: 1px solid var(--rule); border-radius: var(--radius); padding: 28px; width: 420px; display: flex; flex-direction: column; gap: 16px; }
.d-tag { color: var(--accent); margin: 0; }
.d-title { font-size: 24px; margin: 0; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.input { background: transparent; border: 0; border-bottom: 1px solid var(--rule-soft); padding: 8px 0; font-family: var(--font-body); font-size: 14px; color: var(--ink); outline: none; }
.input:focus { border-bottom-color: var(--ink); }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }

@media (max-width: 760px) { .grid { grid-template-columns: 1fr; } }
</style>