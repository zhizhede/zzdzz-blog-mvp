<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const currentPath = computed(() => router.currentRoute.value.path)
const isAdmin = computed(() => userStore.isAdmin)
const inAdminZone = computed(() => currentPath.value.startsWith('/admin'))

const handleCommand = (cmd: string) => {
  if (cmd === 'profile') {
    ElMessage.info('个人信息 — 待实现')
    return
  }
  if (cmd === 'go-client') {
    if (!inAdminZone.value) return
    router.push('/blog')
    return
  }
}

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <el-container class="admin-layout">
    <el-aside width="200px" class="sidebar">
      <div class="logo">zzdzz blog</div>
      <el-menu
        router
        :default-active="$route.path"
        class="menu"
        background-color="#001529"
        text-color="#c9d1d9"
        active-text-color="#409eff"
      >
        <el-menu-item index="/admin/articles">
          <el-icon><Document /></el-icon><span>文章</span>
        </el-menu-item>
        <el-menu-item index="/admin/categories">
          <el-icon><Folder /></el-icon><span>分类</span>
        </el-menu-item>
        <el-menu-item index="/admin/users">
          <el-icon><User /></el-icon><span>用户</span>
        </el-menu-item>
        <el-menu-item index="/admin/ai">
          <el-icon><ChatDotRound /></el-icon><span>AI 对话</span>
        </el-menu-item>
        <el-menu-item index="/blog">
          <el-icon><View /></el-icon><span>前台预览</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="header">
        <span class="welcome">{{ userStore.username }}</span>
        <el-dropdown trigger="click" @command="handleCommand">
          <span class="user-trigger">
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">个人信息</el-dropdown-item>
              <el-dropdown-item v-if="isAdmin" command="go-client" divided>
                前往 client 端
              </el-dropdown-item>
              <el-dropdown-item command="logout" divided @click="handleLogout">
                退出
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.admin-layout { height: 100vh; }
.sidebar { background: #001529; }
.logo {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  padding: 18px;
  text-align: center;
  letter-spacing: 1px;
}
.menu { border-right: none; }
.header {
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}
.welcome { color: #606266; margin-right: auto; padding-left: 8px; }
.user-trigger {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  color: #606266;
}
.user-trigger:hover { background: #f5f7fa; }
.el-main { background: var(--bg); padding: 24px; }
</style>