<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Sunny, Moon, ArrowDown } from '@element-plus/icons-vue'
import { useThemeStore } from '../stores/theme'
import { useUserStore } from '../stores/user'

const route = useRoute()
const router = useRouter()
const theme = useThemeStore()
const user = useUserStore()

const inSpace = computed(() => route.path.startsWith('/space'))
const inAdmin = computed(() => route.path.startsWith('/admin'))

function goHome() {
  router.push('/blog')
}
function toggleTheme() {
  theme.toggle()
}
function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    user.logout()
    ElMessage.success('已退出')
    router.push('/login')
    return
  }
  if (cmd === 'admin') {
    router.push('/admin/articles')
    return
  }
  if (cmd === 'space') {
    router.push('/space/notes')
  }
}
</script>

<template>
  <header class="app-header">
    <div class="container header-inner">
      <a class="brand" @click="goHome">
        <span class="brand-mark">z</span>
        <span class="brand-word">zzdzz</span>
        <span class="brand-tld">blog</span>
      </a>

      <nav class="nav">
        <router-link to="/blog" class="nav-item">阅读</router-link>
        <router-link v-if="user.token" to="/space/notes" class="nav-item">
          空间
        </router-link>
        <router-link v-if="user.isAdmin" to="/admin/articles" class="nav-item">
          后台
        </router-link>
      </nav>

      <div class="right">
        <button class="theme-toggle" :aria-label="theme.theme" @click="toggleTheme">
          <el-icon v-if="theme.theme === 'light'" :size="16"><Moon /></el-icon>
          <el-icon v-else :size="16"><Sunny /></el-icon>
        </button>

        <template v-if="user.token">
          <el-dropdown trigger="click" @command="handleCommand">
            <span class="user-trigger">
              {{ user.username }}
              <el-icon style="margin-left: 4px"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="!inSpace" command="space">个人空间</el-dropdown-item>
                <el-dropdown-item v-if="user.isAdmin && !inAdmin" command="admin" divided>
                  前往后台
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
        <router-link v-else to="/login" class="login-link">登录</router-link>
      </div>
    </div>
    <hr class="rule" />
  </header>
</template>

<style scoped>
.app-header {
  position: sticky;
  top: 0;
  z-index: 50;
  background: var(--bg);
  backdrop-filter: blur(8px);
  border-bottom: 0;
}
.header-inner {
  display: flex;
  align-items: center;
  gap: 24px;
  padding-top: 18px;
  padding-bottom: 18px;
}
.brand {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  border: 0;
  cursor: pointer;
  font-family: var(--font-display);
  color: var(--ink);
}
.brand:hover { color: var(--accent); }
.brand-mark {
  font-size: 22px;
  font-weight: 700;
  color: var(--accent);
}
.brand-word {
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.4px;
}
.brand-tld {
  font-family: var(--font-mono);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  color: var(--ink-mute);
  margin-left: 4px;
}
.nav {
  display: flex;
  gap: 18px;
  margin-left: 16px;
}
.nav-item {
  font-family: var(--font-mono);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  color: var(--ink-mute);
  border: 0;
  padding: 4px 0;
  border-bottom: 1px solid transparent;
}
.nav-item:hover { color: var(--ink); }
.nav-item.router-link-active {
  color: var(--ink);
  border-bottom-color: var(--accent);
}
.right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}
.theme-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--rule-soft);
  background: transparent;
  color: var(--ink-soft);
  border-radius: var(--radius);
  transition: all var(--transition);
}
.theme-toggle:hover {
  color: var(--accent);
  border-color: var(--accent);
}
.user-trigger {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  padding: 6px 10px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.06em;
  color: var(--ink);
  border: 1px solid var(--rule-soft);
}
.user-trigger:hover {
  border-color: var(--ink-mute);
}
.login-link {
  border: 0;
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--ink);
}
.login-link:hover { color: var(--accent); }
.rule {
  border: 0;
  border-top: 1px solid var(--rule);
  margin: 0;
}
</style>
