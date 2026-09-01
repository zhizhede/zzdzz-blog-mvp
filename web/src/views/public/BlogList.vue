<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { articleApi, categoryApi, tagApi, type Article, type Category, type TagWithCount } from '../../api'
import IssueTag from '../../components/IssueTag.vue'
import TagCloud from '../../components/TagCloud.vue'

const route = useRoute()
const router = useRouter()
const articles = ref<Article[]>([])
const categories = ref<Category[]>([])
const tags = ref<TagWithCount[]>([])
const filterCat = ref(0)
const filterTag = ref(0)
const page = ref(1)
const total = ref(0)
const pageSize = 20
const loading = ref(false)

const issueNo = computed(() => {
  const d = new Date()
  const onejan = new Date(d.getFullYear(), 0, 1)
  const week = Math.ceil(((d.getTime() - onejan.getTime()) / 86400000 + onejan.getDay() + 1) / 7)
  return `${d.getFullYear()} · W${WString(week)}`
})
function WString(n: number) {
  return n < 10 ? `0${n}` : String(n)
}

const fetchData = async () => {
  loading.value = true
  try {
    const [a, c, t] = await Promise.all([
      articleApi.list({
        page: page.value,
        size: pageSize,
        category_id: filterCat.value || undefined,
        tag_id: filterTag.value || undefined,
      }),
      categoryApi.list(),
      tagApi.list(),
    ])
    articles.value = a.data.items ?? []
    total.value = a.data.total ?? 0
    categories.value = c.data ?? []
    tags.value = t.data ?? []
  } finally {
    loading.value = false
  }
}

const syncQuery = () => {
  router.replace({
    query: {
      ...(filterCat.value && { cat: String(filterCat.value) }),
      ...(filterTag.value && { tag_id: String(filterTag.value) }),
      ...(page.value > 1 && { page: String(page.value) }),
    },
  })
}

// 服务端已按分类/标签过滤,前端不再二次过滤
const filtered = computed(() => articles.value)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const setFilter = (patch: { cat?: number; tag?: number }) => {
  if (patch.cat !== undefined) filterCat.value = patch.cat
  if (patch.tag !== undefined) filterTag.value = patch.tag
  page.value = 1
  syncQuery()
  fetchData()
}

const goPage = (p: number) => {
  page.value = p
  syncQuery()
  fetchData()
  window.scrollTo({ top: 0 })
}

const categoryName = (id: number) =>
  categories.value.find((c) => c.id === id)?.name || '未分类'
// 没写摘要时取正文前 200 字,去掉 markdown 标记和换行
const summary = (a: Article) =>
  a.summary ||
  a.content
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[#>*`~\[\]!-]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 200)
const readMinutes = (a: Article) => Math.max(1, Math.round(a.content.length / 400))

onMounted(() => {
  filterCat.value = Number(route.query.cat) || 0
  filterTag.value = Number(route.query.tag_id) || 0
  page.value = Number(route.query.page) || 1
  fetchData()
})
</script>

<template>
  <div class="container-narrow home">
    <section class="hero">
      <IssueTag prefix="ISSUE" :text="issueNo" suffix="READING" />
      <h1 class="display hero-title">
        一份 <em>慢</em> 一点的<br />阅读记录。
      </h1>
      <p class="hero-lede">
        这里收录的是 zzdzz 写过的笔记与代码随想。
        一周一篇,每篇尽量写完。
      </p>
      <router-link to="/admin/articles/new" class="publish-btn">发布文章</router-link>
    </section>

    <section class="filters">
      <p class="mono filter-label">FILTER · 分类</p>
      <div class="cat-row">
        <button :class="['cat', !filterCat && 'active']" :aria-pressed="!filterCat" @click="setFilter({ cat: 0 })">全部</button>
        <button
          v-for="c in categories"
          :key="c.id"
          :class="['cat', filterCat === c.id && 'active']"
          :aria-pressed="filterCat === c.id"
          @click="setFilter({ cat: c.id })"
        >{{ c.name }}</button>
      </div>

      <p class="mono filter-label">FILTER · 标签</p>
      <TagCloud :active-tag-id="filterTag" @select="(id) => setFilter({ tag: id ?? 0 })" />
    </section>

    <hr class="hairline" />

    <section class="list" :class="{ loading: loading }">
      <router-link
        v-for="a in filtered"
        :key="a.id"
        :to="`/blog/a/${a.id}`"
        class="post-card"
      >
        <h2 class="post-title display">{{ a.title }}</h2>
        <div class="post-meta">
          <span class="cat-tag">{{ categoryName(a.category_id) }}</span>
          <span class="dot" />
          <span class="mono">{{ new Date(a.created_at).toLocaleDateString() }}</span>
          <span class="dot" />
          <span class="mono">约 {{ readMinutes(a) }} 分钟</span>
          <span class="dot" />
          <span class="mono">{{ a.view_count }} 阅读</span>
          <template v-if="(a.tag_ids ?? []).length">
            <span class="dot" />
            <span
              v-for="tid in (a.tag_ids ?? [])"
              :key="tid"
              class="tag-chip"
            >#{{ tags.find((t) => t.id === tid)?.name || tid }}</span>
          </template>
        </div>
        <p class="post-excerpt">{{ summary(a) }}</p>
      </router-link>
      <div v-if="!filtered.length && !loading" class="empty">这一过滤条件下还没有内容。</div>
      <div v-if="loading" class="empty">加载中…</div>
    </section>

    <div v-if="totalPages > 1" class="pager">
      <button class="pager-btn" :disabled="page <= 1" @click="goPage(page - 1)">← 上一页</button>
      <span class="mono pager-info">第 {{ page }} / {{ totalPages }} 页 · 共 {{ total }} 篇</span>
      <button class="pager-btn" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页 →</button>
    </div>
  </div>
</template>

<style scoped>
.home { padding: 48px 0 80px; }
.hero {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-bottom: 48px;
  margin-bottom: 32px;
  border-bottom: 1px solid var(--rule);
}
.hero-title {
  font-size: 56px;
  line-height: 1;
  margin: 0;
  letter-spacing: -1.2px;
}
.hero-title em {
  font-style: italic;
  color: var(--accent);
  font-weight: 400;
}
.hero-lede {
  color: var(--ink-soft);
  font-size: 16px;
  max-width: 540px;
  margin: 0;
}
.publish-btn {
  align-self: flex-start;
  background: var(--ink);
  color: var(--ink-on-inverse);
  border-radius: var(--radius);
  padding: 10px 22px;
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  text-decoration: none;
  cursor: pointer;
  transition: background var(--transition);
}
.publish-btn:hover { background: var(--accent); color: var(--accent-ink); }
.filters { display: flex; flex-direction: column; gap: 10px; margin-bottom: 32px; }
.filter-label { margin: 12px 0 0; color: var(--ink-mute); }
.cat-row { display: flex; flex-wrap: wrap; gap: 6px; }
.cat {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  background: transparent;
  border: 0;
  font-family: var(--font-body);
  font-size: 14px;
  color: var(--ink-soft);
  padding: 6px 12px;
  border-radius: var(--radius);
  cursor: pointer;
  transition: all var(--transition);
}
.cat:hover { color: var(--ink); }
.cat.active { background: var(--ink); color: var(--ink-on-inverse); }
.hairline { border: 0; border-top: 1px solid var(--rule); }
.list { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.post-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  text-decoration: none;
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 24px;
  cursor: pointer;
  transition: transform var(--transition), border-color var(--transition), box-shadow var(--transition);
}
.post-card:hover {
  border-color: var(--ink);
  transform: translateY(-2px);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
}
.post-title {
  font-size: 24px;
  line-height: 1.25;
  margin: 0;
  color: var(--ink);
}
.post-card:hover .post-title { color: var(--accent); }
.post-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 8px;
  color: var(--ink-mute);
  font-size: 12px;
}
.post-meta .cat-tag {
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-size: 11px;
  color: var(--accent);
}
.post-meta .dot {
  width: 3px;
  height: 3px;
  background: var(--ink-faint);
  border-radius: 50%;
}
.post-meta .mono { font-family: var(--font-mono); }
.post-meta .tag-chip {
  font-family: var(--font-mono);
  color: var(--ink-mute);
}
.post-excerpt {
  color: var(--ink-soft);
  font-size: 14.5px;
  line-height: 1.8;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.empty {
  grid-column: 1 / -1;
  text-align: center;
  padding: 80px 0;
  color: var(--ink-mute);
}
.pager {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 24px;
  margin-top: 16px;
  border-top: 1px solid var(--rule);
}
.pager-btn {
  background: transparent;
  border: 1px solid var(--rule-soft);
  padding: 8px 16px;
  border-radius: var(--radius);
  font-family: var(--font-body);
  font-size: 13px;
  color: var(--ink-soft);
  cursor: pointer;
}
.pager-btn:hover:not(:disabled) { color: var(--ink); border-color: var(--ink-mute); }
.pager-btn:disabled { color: var(--ink-faint); cursor: not-allowed; }
.pager-info { color: var(--ink-mute); font-size: 12px; }
@media (max-width: 760px) {
  .hero-title { font-size: 36px; }
  .list { grid-template-columns: 1fr; }
}
</style>