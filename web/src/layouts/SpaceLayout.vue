<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const currentPath = computed(() => router.currentRoute.value.path)
const isAdmin = computed(() => userStore.isAdmin)

const handleCommand = (cmd: string) => {
  if (cmd === 'profile') {
    router.push('/space/profile')
    return
  }
  if (cmd === 'go-blog') {
    if (currentPath.value.startsWith('/blog')) return
    router.push('/blog')
    return
  }
  if (cmd === 'go-admin') {
    router.push('/admin/articles')
    return
  }
}

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <el-container class="space-layout">
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
        <el-menu-item index="/space/notes">
          <el-icon><Document /></el-icon><span>我的笔记</span>
        </el-menu-item>
        <el-menu-item index="/space/notes/new">
          <el-icon><EditPen /></el-icon><span>写新笔记</span>
        </el-menu-item>
        <el-menu-item index="/space/profile">
          <el-icon><User /></el-icon><span>个人资料</span>
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
              <el-dropdown-item v-if="isAdmin" command="go-admin" divided>
                前往 admin 端
              </el-dropdown-item>
              <el-dropdown-item v-if="!currentPath.startsWith('/blog')" command="go-blog">
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
.space-layout { height: 100vh; }
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