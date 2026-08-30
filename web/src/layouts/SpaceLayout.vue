<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Document, EditPen, User } from '@element-plus/icons-vue'
import AppHeader from '../components/AppHeader.vue'
import IssueTag from '../components/IssueTag.vue'
import { useUserStore } from '../stores/user'

const route = useRoute()
const router = useRouter()
const user = useUserStore()

const today = new Date().toISOString().slice(0, 10).replace(/-/g, '/')
const active = computed(() => route.path)
const isAdmin = computed(() => user.isAdmin)
</script>

<template>
  <div class="space-layout">
    <AppHeader />
    <div class="container space-grid">
      <aside class="side">
        <div class="side-head">
          <IssueTag prefix="SPACE" :text="user.username || 'guest'" :suffix="today" />
          <h2 class="side-title display">个人空间</h2>
          <p class="mono side-hint">这里只放你自己的笔记,可见性你自己定。</p>
        </div>
        <hr class="rule" />
        <nav class="side-nav">
          <button
            :class="['side-item', active.startsWith('/space/notes') && !active.includes('new') && 'active']"
            @click="router.push('/space/notes')"
          >
            <el-icon><Document /></el-icon><span>我的笔记</span>
          </button>
          <button
            :class="['side-item', active.includes('/space/notes/new') && 'active']"
            @click="router.push('/space/notes/new')"
          >
            <el-icon><EditPen /></el-icon><span>写新笔记</span>
          </button>
          <button
            :class="['side-item', active === '/space/profile' && 'active']"
            @click="router.push('/space/profile')"
          >
            <el-icon><User /></el-icon><span>个人资料</span>
          </button>
        </nav>
        <p v-if="isAdmin" class="mono admin-hint">
          你是管理员,可在顶栏下拉前往后台。
        </p>
      </aside>

      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.space-layout {
  min-height: 100vh;
  background: var(--bg);
}
.space-grid {
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
.side-hint {
  font-size: 11px;
  line-height: 1.6;
  color: var(--ink-mute);
  margin: 0;
}
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
.side-item:hover { background: var(--bg-sunken); color: var(--ink); }
.side-item.active {
  background: var(--ink);
  color: var(--ink-on-inverse);
}
.side-item.active :deep(.el-icon) { color: var(--ink-on-inverse); }
.rule { border: 0; border-top: 1px solid var(--rule); margin: 4px 0; }
.admin-hint {
  font-size: 11px;
  color: var(--ink-mute);
  margin: 8px 0 0;
  padding-top: 8px;
  border-top: 1px dashed var(--rule-soft);
}
.content { min-width: 0; }
@media (max-width: 900px) {
  .space-grid { grid-template-columns: 1fr; }
  .side { position: static; }
}
</style>