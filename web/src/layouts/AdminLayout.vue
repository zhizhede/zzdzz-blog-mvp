<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

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
        <el-button text @click="handleLogout">退出</el-button>
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
  gap: 16px;
}
.welcome { color: #606266; margin-right: auto; padding-left: 8px; }
.el-main { background: var(--bg); padding: 24px; }
</style>