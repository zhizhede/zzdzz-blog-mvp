<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, type User } from '../../api'
import { useUserStore } from '../../stores/user'
import IssueTag from '../../components/IssueTag.vue'

const list = ref<User[]>([])
const createDialog = ref(false)
const createForm = ref({ username: '', password: '' })

const passwordDialog = ref(false)
const passwordTarget = ref<User | null>(null)
const passwordForm = ref({ old_password: '', new_password: '', confirm: '' })

const userStore = useUserStore()
const currentUserId = computed(() => userStore.userId)

const fetchList = async () => {
  const res = await userApi.list()
  list.value = res.data ?? []
}

onMounted(fetchList)

const openCreate = () => {
  createForm.value = { username: '', password: '' }
  createDialog.value = true
}

const handleCreate = async () => {
  const u = createForm.value.username.trim()
  const p = createForm.value.password
  if (u.length < 3) {
    ElMessage.warning('用户名至少 3 个字符')
    return
  }
  if (p.length < 6) {
    ElMessage.warning('密码至少 6 位')
    return
  }
  await userApi.create(u, p)
  ElMessage.success('已创建')
  createDialog.value = false
  fetchList()
}

const openPassword = (u: User) => {
  passwordTarget.value = u
  passwordForm.value = { old_password: '', new_password: '', confirm: '' }
  passwordDialog.value = true
}

const handleChangePassword = async () => {
  if (!passwordTarget.value) return
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
  await userApi.changePassword(passwordTarget.value.id, old_password, new_password)
  ElMessage.success('密码已更新')
  passwordDialog.value = false
}

const toggleActive = async (u: User) => {
  const next = !u.is_active
  const action = next ? '启用' : '禁用'
  await ElMessageBox.confirm(`确认${action}用户「${u.username}」?`, '提示', { type: 'warning' })
  await userApi.setActive(u.id, next)
  ElMessage.success(`已${action}`)
  fetchList()
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="ADMIN" text="USERS" :suffix="`${list.length} 人`" />
      <div class="head-row">
        <h1 class="display title">用户</h1>
        <button class="primary-btn" @click="openCreate">
          <span class="mono">＋</span> 新建用户
        </button>
      </div>
    </div>

    <div class="user-list">
      <article v-for="u in list" :key="u.id" class="user-row">
        <div class="user-id">
          <span class="avatar" :class="u.is_admin ? 'admin' : 'reader'">
            {{ u.username.charAt(0).toUpperCase() }}
          </span>
          <div class="who">
            <span class="name">{{ u.username }}</span>
            <span class="mono uid">#{{ u.id }} · {{ u.is_admin ? 'admin' : 'reader' }}</span>
          </div>
        </div>
        <span :class="['vis-pill', u.is_active ? 'vis-public' : 'vis-draft']">
          {{ u.is_active ? '启用' : '禁用' }}
        </span>
        <span class="mono created">{{ new Date(u.created_at).toLocaleDateString() }}</span>
        <div class="actions">
          <button
            class="text-btn"
            :disabled="u.id !== currentUserId"
            @click="openPassword(u)"
          >重置密码</button>
          <button
            class="text-btn"
            :disabled="!u.is_active && u.id === currentUserId"
            @click="toggleActive(u)"
          >{{ u.is_active ? '禁用' : '启用' }}</button>
        </div>
      </article>
      <div v-if="!list.length" class="empty">还没有用户。</div>
    </div>

    <div v-if="createDialog" class="overlay" @click.self="createDialog = false">
      <div class="dialog">
        <p class="mono d-tag">NEW USER</p>
        <h2 class="display d-title">新建用户</h2>
        <label class="field">
          <span class="mono label">USERNAME · 至少 3 字</span>
          <input v-model="createForm.username" class="input" maxlength="64" />
        </label>
        <label class="field">
          <span class="mono label">PASSWORD · 至少 6 位</span>
          <input v-model="createForm.password" type="password" class="input" maxlength="64" />
        </label>
        <div class="d-row">
          <button class="text-btn" @click="createDialog = false">取消</button>
          <button class="primary-btn" @click="handleCreate">创建</button>
        </div>
      </div>
    </div>

    <div v-if="passwordDialog" class="overlay" @click.self="passwordDialog = false">
      <div class="dialog">
        <p class="mono d-tag">RESET PASSWORD</p>
        <h2 class="display d-title">重置密码 · {{ passwordTarget?.username }}</h2>
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
          <button class="primary-btn" @click="handleChangePassword">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 24px; padding-bottom: 64px; }
.page-head { display: flex; flex-direction: column; gap: 12px; }
.head-row { display: flex; justify-content: space-between; align-items: baseline; }
.title { font-size: 36px; line-height: 1; margin: 0; letter-spacing: -0.8px; }
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
.text-btn:disabled { color: var(--ink-faint); cursor: not-allowed; border-bottom-color: transparent; }

.user-list { display: flex; flex-direction: column; gap: 0; }
.user-row {
  display: grid;
  grid-template-columns: 1fr 90px 120px 200px;
  gap: 16px;
  align-items: center;
  padding: 16px 12px;
  border-bottom: 1px solid var(--rule-soft);
}
.user-id { display: flex; gap: 12px; align-items: center; min-width: 0; }
.avatar {
  width: 36px; height: 36px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  border: 1px solid var(--rule-soft);
}
.avatar.admin { background: var(--accent); color: var(--accent-ink); border-color: var(--accent); }
.avatar.reader { background: var(--bg-sunken); color: var(--ink); }
.who { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.name { font-family: var(--font-display); font-size: 15px; font-weight: 500; color: var(--ink); }
.uid { font-size: 11px; color: var(--ink-mute); }
.created { color: var(--ink-mute); font-size: 12px; }
.actions { display: flex; gap: 12px; justify-content: flex-end; }

.vis-pill {
  font-family: var(--font-mono);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 3px 8px;
  border-radius: var(--radius);
  display: inline-block;
  width: fit-content;
}
.vis-public { background: var(--accent); color: var(--accent-ink); }
.vis-draft { background: transparent; color: var(--ink-mute); border: 1px dashed var(--rule-soft); }

.empty { padding: 60px 0; text-align: center; color: var(--ink-mute); }

.overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 100; }
.dialog { background: var(--bg); border: 1px solid var(--rule); border-radius: var(--radius); padding: 28px; width: 420px; display: flex; flex-direction: column; gap: 16px; }
.d-tag { color: var(--accent); margin: 0; }
.d-title { font-size: 24px; margin: 0; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.input { background: transparent; border: 0; border-bottom: 1px solid var(--rule-soft); padding: 8px 0; font-family: var(--font-body); font-size: 14px; color: var(--ink); outline: none; }
.input:focus { border-bottom-color: var(--ink); }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }
</style>