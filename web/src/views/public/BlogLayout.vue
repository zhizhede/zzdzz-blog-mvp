<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../../stores/user'

const router = useRouter()
const userStore = useUserStore()

const isAdmin = computed(() => userStore.isAdmin)
const loggedIn = computed(() => !!userStore.token)

const handleCommand = (cmd: string) => {
  if (cmd === 'profile') {
    router.push('/space/profile')
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

const handleLogin = () => {
  router.push('/login')
}
</script>

<template>
  <div class="blog">
    <header class="blog-header">
      <div class="blog-header-inner">
        <h1 class="brand" @click="router.push('/blog')">zzdzz blog</h1>
        <div class="right">
          <template v-if="loggedIn">
            <el-dropdown trigger="click" @command="handleCommand">
              <span class="user-trigger">
                {{ userStore.username }}<el-icon style="margin-left:4px"><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="profile">个人信息</el-dropdown-item>
                  <el-dropdown-item
                    v-if="isAdmin"
                    command="go-admin"
                    divided
                  >
                    前往 admin 端
                  </el-dropdown-item>
                  <el-dropdown-item command="logout" divided @click="handleLogout">
                    退出
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <a class="login-link" @click="handleLogin">登录</a>
          </template>
        </div>
      </div>
    </header>
    <main class="blog-main">
      <router-view />
    </main>
    <footer class="blog-footer">© 2026 zzdzz blog · powered by Go + Vue3</footer>
  </div>
</template>

<style scoped>
.blog { min-height: 100vh; display: flex; flex-direction: column; }
.blog-header {
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.blog-header-inner {
  max-width: 800px; margin: 0 auto; padding: 16px 24px;
  display: flex; align-items: center; justify-content: space-between;
}
.brand { margin: 0; font-size: 20px; cursor: pointer; letter-spacing: 1px; }
.right { display: flex; align-items: center; gap: 12px; }
.user-trigger {
  display: inline-flex; align-items: center; cursor: pointer;
  padding: 4px 10px; border-radius: 4px; color: #606266;
}
.user-trigger:hover { background: #f5f7fa; }
.login-link { color: #409eff; cursor: pointer; }
.blog-main { flex: 1; max-width: 800px; margin: 0 auto; padding: 32px 24px; width: 100%; box-sizing: border-box; }
.blog-footer { text-align: center; padding: 20px; color: #909399; border-top: 1px solid #ebeef5; background: #fff; }
a { color: #409eff; text-decoration: none; }
</style>