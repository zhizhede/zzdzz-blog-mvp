<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { articleApi, categoryApi, type Article, type Category } from '../../api'
import { useUserStore } from '../../stores/user'
import IssueTag from '../../components/IssueTag.vue'

const router = useRouter()
const userStore = useUserStore()

const myUserId = computed(() => userStore.userId)
const list = ref<Article[]>([])
const categories = ref<Category[]>([])
const total = ref(0)
const loading = ref(false)
const query = ref({ page: 1, size: 10, category_id: 0, q: '' })

const fetchList = async () => {
  if (!myUserId.value) return
  loading.value = true
  try {
    const res = await articleApi.list({
      page: query.value.page,
      size: query.value.size,
      category_id: query.value.category_id || undefined,
      q: query.value.q || undefined,
      author_id: myUserId.value,
    })
    list.value = res.data.items ?? []
    total.value = res.data.total ?? 0
  } finally {
    loading.value = false
  }
}

const fetchCategories = async () => {
  const res = await categoryApi.list()
  categories.value = res.data ?? []
}

const categoryName = (id: number) =>
  categories.value.find((c) => c.id === id)?.name || '-'

const visLabel = (v: string) =>
  v === 'public' ? '公开' : v === 'private' ? '仅自己' : v === 'draft' ? '草稿' : v || '公开'

const visClass = (v: string) =>
  v === 'public' ? 'vis-public' : v === 'private' ? 'vis-private' : 'vis-draft'

const handleDelete = async (row: Article) => {
  await ElMessageBox.confirm(`确认删除「${row.title}」?`, '提示', { type: 'warning' })
  await articleApi.remove(row.id)
  fetchList()
}

onMounted(() => {
  fetchCategories()
  fetchList()
})
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="SPACE" text="NOTES" :suffix="`${total} 篇`" />
      <div class="head-row">
        <h1 class="display title">我的笔记</h1>
        <button class="primary-btn" @click="router.push('/space/notes/new')">
          <span class="mono">＋</span> 写新笔记
        </button>
      </div>
    </div>

    <div class="filters">
      <p class="mono filter-label">FILTER · 分类</p>
      <div class="cat-row">
        <button :class="['cat', !query.category_id && 'active']" @click="query.category_id = 0; fetchList()">全部</button>
        <button
          v-for="c in categories"
          :key="c.id"
          :class="['cat', query.category_id === c.id && 'active']"
          @click="query.category_id = c.id; fetchList()"
        >{{ c.name }}</button>
      </div>

      <p class="mono filter-label">FILTER · 搜索</p>
      <div class="search-row">
        <input v-model="query.q" class="input" placeholder="按标题 / 摘要搜索" @keyup.enter="fetchList" />
        <button class="text-btn" @click="fetchList">搜索</button>
      </div>
    </div>

    <div class="list">
      <article v-for="row in list" :key="row.id" class="note-card" @click="router.push(`/space/notes/${row.id}/edit`)">
        <div class="note-meta">
          <span class="mono">#{{ row.id }}</span>
          <span :class="['vis-pill', visClass(row.visibility)]">{{ visLabel(row.visibility) }}</span>
          <span class="mono date">{{ new Date(row.created_at).toLocaleDateString() }}</span>
        </div>
        <h2 class="note-title display">{{ row.title }}</h2>
        <p class="note-excerpt">{{ row.summary || row.content.slice(0, 120).replace(/\n/g, ' ') }}</p>
        <div class="note-foot">
          <span class="mono cat-tag">{{ categoryName(row.category_id) }}</span>
          <button class="text-btn danger" @click.stop="handleDelete(row)">删除</button>
        </div>
      </article>
      <div v-if="loading" class="empty">加载中…</div>
      <div v-else-if="!list.length" class="empty">还没有笔记。</div>
    </div>

    <div class="pager">
      <button class="text-btn" :disabled="query.page <= 1" @click="query.page--; fetchList()">← 上一页</button>
      <span class="mono pager-info">第 {{ query.page }} 页 · 共 {{ total }} 条</span>
      <button class="text-btn" :disabled="query.page * query.size >= total" @click="query.page++; fetchList()">下一页 →</button>
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
.text-btn.danger:hover { color: var(--danger); border-bottom-color: var(--danger); }
.text-btn:disabled { color: var(--ink-faint); cursor: not-allowed; }

.filters { display: flex; flex-direction: column; gap: 10px; padding-bottom: 16px; border-bottom: 1px solid var(--rule); }
.filter-label { margin: 8px 0 0; color: var(--ink-mute); }
.cat-row { display: flex; flex-wrap: wrap; gap: 6px; }
.cat {
  background: transparent;
  border: 1px solid var(--rule-soft);
  padding: 4px 12px;
  border-radius: var(--radius);
  font-family: var(--font-body);
  font-size: 13px;
  color: var(--ink-soft);
  cursor: pointer;
}
.cat:hover { color: var(--ink); border-color: var(--ink-mute); }
.cat.active { background: var(--ink); color: var(--ink-on-inverse); border-color: var(--ink); }
.search-row { display: flex; gap: 12px; align-items: center; }
.input {
  flex: 1;
  max-width: 360px;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 6px 0;
  font-family: var(--font-body);
  font-size: 14px;
  color: var(--ink);
  outline: none;
}

.list { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.note-card {
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 20px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: all var(--transition);
}
.note-card:hover { border-color: var(--ink); transform: translateY(-2px); }
.note-meta { display: flex; gap: 10px; align-items: center; color: var(--ink-mute); font-size: 11px; }
.note-meta .date { margin-left: auto; }
.note-title { font-size: 20px; line-height: 1.2; margin: 0; }
.note-card:hover .note-title { color: var(--accent); }
.note-excerpt { color: var(--ink-soft); font-size: 13px; line-height: 1.6; margin: 0; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.note-foot { display: flex; justify-content: space-between; align-items: center; padding-top: 8px; border-top: 1px solid var(--rule-soft); }
.cat-tag { color: var(--accent); font-size: 11px; text-transform: uppercase; letter-spacing: 0.1em; }

.vis-pill {
  font-family: var(--font-mono);
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 2px 8px;
  border-radius: var(--radius);
}
.vis-public { background: var(--accent); color: var(--accent-ink); }
.vis-private { background: var(--bg-sunken); color: var(--ink-soft); border: 1px solid var(--rule); }
.vis-draft { background: transparent; color: var(--ink-mute); border: 1px dashed var(--rule-soft); }

.empty { grid-column: 1 / -1; text-align: center; padding: 60px 0; color: var(--ink-mute); }
.pager { display: flex; justify-content: space-between; align-items: center; padding-top: 16px; }
.pager-info { color: var(--ink-mute); font-size: 12px; }

@media (max-width: 760px) {
  .list { grid-template-columns: 1fr; }
}
</style>