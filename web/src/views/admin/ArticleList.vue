<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { articleApi, categoryApi, type Article, type Category } from '../../api'
import IssueTag from '../../components/IssueTag.vue'

const router = useRouter()
const list = ref<Article[]>([])
const categories = ref<Category[]>([])
const total = ref(0)
const loading = ref(false)
const query = ref({ page: 1, size: 10, category_id: 0, q: '' })

const fetchList = async () => {
  loading.value = true
  try {
    const res = await articleApi.list({
      page: query.value.page,
      size: query.value.size,
      category_id: query.value.category_id || undefined,
      q: query.value.q || undefined,
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

const visibleList = computed(() => list.value)

onMounted(() => {
  fetchCategories()
  fetchList()
})
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="ADMIN" text="ARTICLES" :suffix="`${total} 篇`" />
      <div class="head-row">
        <h1 class="display title">文章</h1>
        <button class="primary-btn" @click="router.push('/admin/articles/new')">
          <span class="mono">＋</span> 写新文章
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
        <input
          v-model="query.q"
          class="input"
          placeholder="按标题 / 摘要搜索"
          @keyup.enter="fetchList"
        />
        <button class="text-btn" @click="fetchList">搜索</button>
      </div>
    </div>

    <div class="table-wrap">
      <div class="row row-head">
        <span class="cell c-id">ID</span>
        <span class="cell c-title">标题</span>
        <span class="cell c-cat">分类</span>
        <span class="cell c-vis">可见性</span>
        <span class="cell c-stat">阅读</span>
        <span class="cell c-date">创建时间</span>
        <span class="cell c-act">操作</span>
      </div>
      <div v-if="loading" class="loading">加载中…</div>
      <div v-else-if="!visibleList.length" class="empty">还没有文章。</div>
      <div
        v-for="row in visibleList"
        :key="row.id"
        class="row"
      >
        <span class="cell c-id mono">#{{ row.id }}</span>
        <span class="cell c-title">{{ row.title }}</span>
        <span class="cell c-cat">{{ categoryName(row.category_id) }}</span>
        <span :class="['cell c-vis vis-pill', visClass(row.visibility)]">
          {{ visLabel(row.visibility) }}
        </span>
        <span class="cell c-stat mono">{{ row.view_count }}</span>
        <span class="cell c-date mono">{{ new Date(row.created_at).toLocaleDateString() }}</span>
        <span class="cell c-act">
          <button class="text-btn" @click="router.push(`/admin/articles/${row.id}/edit`)">编辑</button>
          <button class="text-btn danger" @click="handleDelete(row)">删除</button>
        </span>
      </div>
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
.input:focus { border-bottom-color: var(--ink); }

.table-wrap { display: flex; flex-direction: column; gap: 0; }
.row {
  display: grid;
  grid-template-columns: 60px 1fr 100px 90px 80px 130px 160px;
  gap: 12px;
  padding: 14px 12px;
  border-bottom: 1px solid var(--rule-soft);
  align-items: center;
  font-size: 14px;
  color: var(--ink);
}
.row-head { background: var(--bg-sunken); border-bottom: 1px solid var(--rule); font-family: var(--font-mono); font-size: 11px; text-transform: uppercase; letter-spacing: 0.12em; color: var(--ink-mute); }
.cell { min-width: 0; }
.c-title { font-family: var(--font-display); font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.c-date { color: var(--ink-mute); font-size: 12px; }
.row:hover .c-title { color: var(--accent); }

.vis-pill {
  font-family: var(--font-mono);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 3px 8px;
  border-radius: var(--radius);
  display: inline-block;
  border: 1px solid transparent;
}
.vis-public { background: var(--accent); color: var(--accent-ink); }
.vis-private { background: var(--bg-sunken); color: var(--ink-soft); border-color: var(--rule); }
.vis-draft { background: transparent; color: var(--ink-mute); border-color: var(--rule-soft); border-style: dashed; }

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

.c-act { display: flex; gap: 12px; }

.loading, .empty { padding: 40px; text-align: center; color: var(--ink-mute); }
.pager { display: flex; justify-content: space-between; align-items: center; padding-top: 16px; }
.pager-info { color: var(--ink-mute); font-size: 12px; }
</style>