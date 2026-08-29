<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, type User } from '../../api'
import { useUserStore } from '../../stores/user'

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
  list.value = res.data
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
  await ElMessageBox.confirm(
    `确认${action}用户「${u.username}」?`,
    '提示',
    { type: 'warning' },
  )
  await userApi.setActive(u.id, next)
  ElMessage.success(`已${action}`)
  fetchList()
}
</script>

<template>
  <div class="page-container">
    <div class="toolbar">
      <h2>用户</h2>
      <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>新建用户</el-button>
    </div>

    <el-card>
      <el-table :data="list" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_active" type="success">启用</el-tag>
            <el-tag v-else type="danger">禁用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button
              text
              type="primary"
              :disabled="row.id !== currentUserId"
              @click="openPassword(row)"
            >
              重置密码
            </el-button>
            <el-button
              :type="row.is_active ? 'danger' : 'success'"
              text
              :disabled="!row.is_active && row.id === currentUserId"
              @click="toggleActive(row)"
            >
              {{ row.is_active ? '禁用' : '启用' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="createDialog" title="新建用户" width="420px">
      <el-form label-width="80px">
        <el-form-item label="用户名" required>
          <el-input v-model="createForm.username" maxlength="64" placeholder="至少 3 个字符" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input
            v-model="createForm.password"
            type="password"
            show-password
            maxlength="64"
            placeholder="至少 6 位"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="passwordDialog" title="重置密码" width="420px">
      <el-form v-if="passwordTarget" label-width="100px">
        <el-form-item label="用户">
          <span>{{ passwordTarget.username }}</span>
        </el-form-item>
        <el-form-item label="旧密码" required>
          <el-input
            v-model="passwordForm.old_password"
            type="password"
            show-password
            placeholder="自己的账号要输入当前密码"
          />
        </el-form-item>
        <el-form-item label="新密码" required>
          <el-input
            v-model="passwordForm.new_password"
            type="password"
            show-password
            placeholder="至少 6 位"
          />
        </el-form-item>
        <el-form-item label="确认新密码" required>
          <el-input
            v-model="passwordForm.confirm"
            type="password"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialog = false">取消</el-button>
        <el-button type="primary" @click="handleChangePassword">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.toolbar h2 { margin: 0; }
</style>