<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { authApi } from '../../api'
import { useUserStore } from '../../stores/user'

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
  <div class="page-container">
    <h2>个人资料</h2>

    <el-card>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="用户名">{{ userStore.username }}</el-descriptions-item>
        <el-descriptions-item label="用户 ID">{{ userStore.userId }}</el-descriptions-item>
        <el-descriptions-item label="角色">
          <el-tag v-if="userStore.isAdmin" type="success">管理员</el-tag>
          <el-tag v-else type="info">普通用户</el-tag>
        </el-descriptions-item>
      </el-descriptions>

      <div class="actions">
        <el-button type="primary" @click="passwordDialog = true">重置密码</el-button>
      </div>
    </el-card>

    <el-dialog v-model="passwordDialog" title="重置密码" width="420px">
      <el-form label-width="100px">
        <el-form-item label="旧密码" required>
          <el-input
            v-model="passwordForm.old_password"
            type="password"
            show-password
            placeholder="你的当前密码"
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
        <el-button type="primary" :loading="saving" @click="handleChangePassword">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container h2 { margin-top: 0; }
.actions { margin-top: 16px; display: flex; justify-content: flex-end; }
</style>