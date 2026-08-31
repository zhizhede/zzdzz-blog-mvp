<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { authApi } from '../api'
import { useUserStore } from '../stores/user'
import IssueTag from '../components/IssueTag.vue'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const res = await authApi.login(form.value.username, form.value.password)
    userStore.setAuth(res.data.token, res.data.user.id, res.data.user.username, !!res.data.user.is_admin)
    ElMessage.success('登录成功')
    const redirect = (route.query.redirect as string) || '/admin/articles'
    router.push(redirect)
  } catch {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-card">
      <IssueTag prefix="ISSUE" text="01" suffix="ACCESS" />
      <h1 class="display title">zzdzz <em>blog</em></h1>
      <p class="lede">登录到管理后台或个人空间。</p>

      <form class="form" @submit.prevent="handleLogin">
        <label class="field">
          <span class="mono label">USERNAME</span>
          <input
            v-model="form.username"
            class="input"
            placeholder="用户名"
            autocomplete="username"
          />
        </label>
        <label class="field">
          <span class="mono label">PASSWORD</span>
          <input
            v-model="form.password"
            type="password"
            class="input"
            placeholder="密码"
            autocomplete="current-password"
            @keyup.enter="handleLogin"
          />
        </label>
        <button class="primary-btn" :disabled="loading" @click.prevent="handleLogin">
          <span v-if="loading" class="mono">…</span>
          <span v-else>登 录</span>
        </button>
      </form>

    </div>
  </div>
</template>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  padding: 24px;
}
.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--bg-elev);
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  padding: 40px 36px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.title {
  font-size: 48px;
  line-height: 1;
  margin: 8px 0 0;
  letter-spacing: -1px;
}
.title em {
  font-style: italic;
  color: var(--accent);
  font-weight: 400;
}
.lede {
  color: var(--ink-soft);
  font-size: 14px;
  margin: 0 0 16px;
}
.form { display: flex; flex-direction: column; gap: 18px; }
.field { display: flex; flex-direction: column; gap: 8px; }
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.input {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 10px 0;
  font-family: var(--font-body);
  font-size: 16px;
  color: var(--ink);
  outline: none;
  width: 100%;
}
.input:focus { border-bottom-color: var(--ink); }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 12px 24px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 13px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  cursor: pointer;
  margin-top: 8px;
}
.primary-btn:hover { background: var(--accent); }
.primary-btn:disabled { background: var(--ink-faint); cursor: not-allowed; }
</style>