<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Document,
  Folder,
  User,
  ChatDotRound,
} from '@element-plus/icons-vue'
import AppHeader from '../components/AppHeader.vue'
import IssueTag from '../components/IssueTag.vue'

const route = useRoute()
const router = useRouter()

const active = computed(() => route.path)
const today = new Date().toISOString().slice(0, 10).replace(/-/g, '/')
</script>

<template>
  <div class="admin-layout">
    <AppHeader />
    <div class="container admin-grid">
      <aside class="side">
        <div class="side-head">
          <IssueTag prefix="ADMIN" text="DESK" :suffix="today" />
          <h2 class="side-title display">控制台</h2>
        </div>
        <nav class="side-nav">
          <button
            :class="['side-item', active.startsWith('/admin/articles') && 'active']"
            @click="router.push('/admin/articles')"
          >
            <el-icon><Document /></el-icon><span>文章</span>
            <em class="mono">all</em>
          </button>
          <button
            :class="['side-item', active === '/admin/categories' && 'active']"
            @click="router.push('/admin/categories')"
          >
            <el-icon><Folder /></el-icon><span>分类</span>
            <em class="mono">tax</em>
          </button>
          <button
            :class="['side-item', active === '/admin/users' && 'active']"
            @click="router.push('/admin/users')"
          >
            <el-icon><User /></el-icon><span>用户</span>
            <em class="mono">acl</em>
          </button>
          <button
            :class="['side-item', active === '/admin/ai' && 'active']"
            @click="router.push('/admin/ai')"
          >
            <el-icon><ChatDotRound /></el-icon><span>AI 对话</span>
            <em class="mono">llm</em>
          </button>
        </nav>

        <hr class="rule" />
        <p class="mono side-hint">
          后台管理 · 内容、用户、AI 会话
        </p>
      </aside>

      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.admin-layout {
  min-height: 100vh;
  background: var(--bg);
}
.admin-grid {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 32px;
  padding-top: 32px;
  padding-bottom: 64px;
}
.side {
  position: sticky;
  top: 92px;
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.side-head { display: flex; flex-direction: column; gap: 10px; }
.side-title { font-size: 24px; margin: 0; }
.side-nav { display: flex; flex-direction: column; gap: 2px; }
.side-item {
  display: flex;
  align-items: center;
  gap: 10px;
  background: transparent;
  border: 0;
  text-align: left;
  font-family: var(--font-body);
  font-size: 14px;
  color: var(--ink-soft);
  padding: 10px 12px;
  border-radius: var(--radius);
  cursor: pointer;
  transition: all var(--transition);
}
.side-item em {
  margin-left: auto;
  font-style: normal;
  color: var(--ink-faint);
  font-size: 10px;
  letter-spacing: 0.16em;
}
.side-item:hover { background: var(--bg-sunken); color: var(--ink); }
.side-item.active {
  background: var(--ink);
  color: var(--ink-on-inverse);
}
.side-item.active em { color: var(--ink-on-inverse); opacity: 0.5; }
.side-item.active :deep(.el-icon) { color: var(--ink-on-inverse); }
.rule { border: 0; border-top: 1px solid var(--rule); margin: 8px 0; }
.side-hint {
  font-size: 11px;
  line-height: 1.6;
  color: var(--ink-mute);
  margin: 0;
}
.content { min-width: 0; }
@media (max-width: 900px) {
  .admin-grid { grid-template-columns: 1fr; }
  .side { position: static; }
}
</style>