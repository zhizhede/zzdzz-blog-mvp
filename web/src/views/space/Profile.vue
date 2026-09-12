<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../../stores/user'
import IssueTag from '../../components/IssueTag.vue'
import ChangePasswordDialog from '../../components/ChangePasswordDialog.vue'

const userStore = useUserStore()

const pwdDialog = ref(false)

// 站点是 HTTP(IP 直访)时 navigator.clipboard 不存在, 降级 execCommand
const copyText = async (text: string) => {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text)
    return
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  document.execCommand('copy')
  document.body.removeChild(ta)
}

const copied = ref(false)
const copyUuid = async () => {
  if (!userStore.userUuid) return
  try {
    await copyText(userStore.userUuid)
    copied.value = true
    ElMessage.success('UUID 已复制')
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    ElMessage.error('复制失败, 请手动选择复制')
  }
}

// 老会话的 localStorage 里可能还没有 uuid(登录早于 0013 上线), 补拉一次
onMounted(() => {
  if (!userStore.userUuid) userStore.refresh()
})
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
            <button
              class="uuid-row"
              :title="userStore.userUuid ? '点击复制' : ''"
              @click="copyUuid"
            >
              <span class="mono uuid-label">UUID</span>
              <span class="mono uuid-value">{{ userStore.userUuid || '获取中…' }}</span>
              <span v-if="userStore.userUuid" class="mono uuid-copy">{{ copied ? '已复制' : '复制' }}</span>
            </button>
          </div>
        </div>
        <div class="actions">
          <button class="primary-btn" @click="pwdDialog = true">
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

    <ChangePasswordDialog v-model="pwdDialog" />
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
.uuid-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
  background: transparent;
  border: 0;
  border-bottom: 1px dashed var(--rule-soft);
  padding: 4px 0 6px;
  text-align: left;
  cursor: pointer;
  max-width: 100%;
}
.uuid-row:hover .uuid-copy { color: var(--accent); }
.uuid-label {
  font-size: 10px;
  color: var(--ink-mute);
  letter-spacing: 0.16em;
}
.uuid-value {
  font-size: 12px;
  color: var(--ink-soft);
  word-break: break-all;
}
.uuid-copy {
  font-size: 11px;
  color: var(--ink-mute);
  margin-left: auto;
  transition: color var(--transition);
}
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

@media (max-width: 760px) { .grid { grid-template-columns: 1fr; } }
</style>
