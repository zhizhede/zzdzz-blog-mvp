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

// mode 由 BlogLayout 的「注册」入口经 query 初始化, 页内可随时切换
const mode = ref<'login' | 'register'>(route.query.mode === 'register' ? 'register' : 'login')
const form = ref({ username: '', password: '', confirmPassword: '', passwordHint: '' })
const loading = ref(false)

// 忘记密码: 按用户名查注册时留的密码提示(公开接口)
const hintLookup = ref({ open: false, username: '', result: '', tried: false, loading: false })
const lookupHint = async () => {
  const uname = hintLookup.value.username.trim()
  if (!uname) {
    ElMessage.warning('请输入用户名')
    return
  }
  hintLookup.value.loading = true
  try {
    const res = await authApi.passwordHint(uname)
    hintLookup.value.result = res.data.password_hint
    hintLookup.value.tried = true
  } catch {
    // 拦截器已提示
  } finally {
    hintLookup.value.loading = false
  }
}

const redirectAfterAuth = () => {
  const redirect = route.query.redirect as string
  if (redirect) return redirect
  // 默认分流: admin 回后台, 普通用户回公开博客首页
  return userStore.isAdmin ? '/admin/articles' : '/blog'
}

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const res = await authApi.login(form.value.username, form.value.password)
    userStore.setAuth(
      res.data.token,
      res.data.user.id,
      res.data.user.username,
      !!res.data.user.is_admin,
      res.data.user.uuid || ''
    )
    ElMessage.success('登录成功')
    router.push(redirectAfterAuth())
  } catch {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

const handleRegister = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  if (form.value.password.length < 6) {
    ElMessage.warning('密码至少 6 位')
    return
  }
  if (form.value.confirmPassword !== form.value.password) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  loading.value = true
  try {
    const res = await authApi.register(form.value.username.trim(), form.value.password, form.value.passwordHint.trim())
    userStore.setAuth(
      res.data.token,
      res.data.user.id,
      res.data.user.username,
      !!res.data.user.is_admin,
      res.data.user.uuid || ''
    )
    ElMessage.success('注册成功, 已自动登录')
    router.push(redirectAfterAuth())
  } catch {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

const submit = () => (mode.value === 'login' ? handleLogin() : handleRegister())
const switchMode = (m: 'login' | 'register') => {
  mode.value = m
  if (m === 'login') form.value.confirmPassword = ''
  else form.value.passwordHint = ''
  hintLookup.value = { open: false, username: '', result: '', tried: false, loading: false }
  // 让 /login?mode=register 与页内状态保持一致(前进后退均可复现)
  router.replace({ path: '/login', query: m === 'register' ? { mode: 'register' } : {} })
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-card">
      <IssueTag prefix="ISSUE" text="01" suffix="ACCESS" />
      <h1 class="display title">zzdzz <em>blog</em></h1>
      <p class="lede">
        {{ mode === 'login' ? '登录到管理后台或个人空间。' : '用户名全系统唯一, 输入用户名和密码即可注册, 成功后自动登录。' }}
      </p>

      <form class="form" @submit.prevent="submit">
        <label class="field">
          <span class="mono label">USERNAME</span>
          <input
            v-model="form.username"
            class="input"
            placeholder="用户名"
            autocomplete="username"
            maxlength="64"
          />
        </label>
        <label class="field">
          <span class="mono label">PASSWORD</span>
          <input
            v-model="form.password"
            type="password"
            class="input"
            :placeholder="mode === 'register' ? '密码(至少 6 位)' : '密码'"
            :autocomplete="mode === 'register' ? 'new-password' : 'current-password'"
            maxlength="64"
            @keyup.enter="submit"
          />
        </label>
        <label v-if="mode === 'register'" class="field">
          <span class="mono label">CONFIRM PASSWORD</span>
          <input
            v-model="form.confirmPassword"
            type="password"
            class="input"
            placeholder="再输一遍密码"
            autocomplete="new-password"
            maxlength="64"
            @keyup.enter="submit"
          />
        </label>
        <label v-if="mode === 'register'" class="field">
          <span class="mono label">PASSWORD HINT · OPTIONAL</span>
          <input
            v-model="form.passwordHint"
            class="input"
            placeholder="密码提示, 想不起来时可以回来查看(选填)"
            maxlength="255"
          />
        </label>
        <button class="primary-btn" :disabled="loading" @click.prevent="submit">
          <span v-if="loading" class="mono">…</span>
          <span v-else>{{ mode === 'login' ? '登 录' : '注 册' }}</span>
        </button>
      </form>

      <div v-if="mode === 'login' && hintLookup.open" class="hint-lookup">
        <div class="hint-row">
          <input
            v-model="hintLookup.username"
            class="input"
            placeholder="输入注册时的用户名"
            maxlength="64"
            @keyup.enter="lookupHint"
          />
          <button class="ghost-btn" :disabled="hintLookup.loading" @click.prevent="lookupHint">查提示</button>
        </div>
        <p v-if="hintLookup.result" class="hint-result">
          密码提示: <strong>{{ hintLookup.result }}</strong>
        </p>
        <p v-else-if="!hintLookup.loading && hintLookup.tried" class="hint-result muted">
          该用户名没有设置密码提示
        </p>
      </div>

      <div class="mode-links">
        <template v-if="mode === 'login'">
          <a class="link" @click="hintLookup.open = !hintLookup.open">忘记密码? 看看密码提示 →</a>
          <a class="link" @click="switchMode('register')">没有账号? 注册一个 →</a>
        </template>
        <template v-else>
          <a class="link" @click="switchMode('login')">已有账号? 去登录 →</a>
        </template>
        <a class="link ghost" @click="router.push('/blog')">先去逛逛博客 →</a>
      </div>
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
  font-size: 60px;
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
  font-size: 17.5px;
  margin: 0 0 16px;
}
.form { display: flex; flex-direction: column; gap: 18px; }
.field { display: flex; flex-direction: column; gap: 8px; }
.label { color: var(--ink-mute); font-size: 13.75px; text-transform: uppercase; letter-spacing: 0.16em; }
.input {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 10px 0;
  font-family: var(--font-body);
  font-size: 20px;
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
  font-size: 16.25px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  cursor: pointer;
  margin-top: 8px;
}
.primary-btn:hover { background: var(--accent); }
.primary-btn:disabled { background: var(--ink-faint); cursor: not-allowed; }
.mode-links {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 16.25px;
}
.link { color: var(--accent); cursor: pointer; }
.link.ghost { color: var(--ink-mute); }
.hint-lookup {
  border: 1px dashed var(--rule);
  border-radius: var(--radius);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.hint-row { display: flex; gap: 10px; }
.hint-row .input { flex: 1; }
.ghost-btn {
  background: transparent;
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  color: var(--ink);
  font-family: var(--font-mono);
  font-size: 15px;
  padding: 0 14px;
  cursor: pointer;
  white-space: nowrap;
}
.ghost-btn:hover { border-color: var(--accent); color: var(--accent); }
.ghost-btn:disabled { color: var(--ink-faint); cursor: not-allowed; }
.hint-result { margin: 0; font-size: 16.25px; color: var(--ink); }
.hint-result.muted { color: var(--ink-mute); }
</style>
