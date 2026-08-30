<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
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
  const [a, c, t] = await Promise.all([
    articleApi.list({
      page: 1,
      size: 50,
      category_id: filterCat.value || undefined,
      tag_id: filterTag.value || undefined,
    }),
    categoryApi.list(),
    tagApi.list(),
  ])
  articles.value = a.data.items ?? []
  categories.value = c.data ?? []
  tags.value = t.data ?? []
}

const filtered = computed(() => {
  let r = articles.value
  if (filterCat.value) r = r.filter((a) => a.category_id === filterCat.value)
  if (filterTag.value) r = r.filter((a) => (a.tag_ids ?? []).includes(filterTag.value))
  return r
})

const categoryName = (id: number) =>
  categories.value.find((c) => c.id === id)?.name || '未分类'
const summary = (a: Article) => a.summary || a.content.slice(0, 120).replace(/\n/g, ' ')

onMounted(() => {
  const q = route.query.tag_id
  if (q) filterTag.value = Number(q) || 0
  fetchData()
})
watch([filterCat, filterTag], () => fetchData())
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
    </section>

    <section class="filters">
      <p class="mono filter-label">FILTER · 分类</p>
      <div class="cat-row">
        <button :class="['cat', !filterCat && 'active']" @click="filterCat = 0">全部</button>
        <button
          v-for="c in categories"
          :key="c.id"
          :class="['cat', filterCat === c.id && 'active']"
          @click="filterCat = c.id"
        >{{ c.name }}</button>
      </div>

      <p class="mono filter-label">FILTER · 标签</p>
      <TagCloud :active-tag-id="filterTag" @select="(id) => (filterTag = id ?? 0)" />
    </section>

    <hr class="hairline" />

    <section class="list">
      <article
        v-for="a in filtered"
        :key="a.id"
        class="post-card"
        @click="router.push(`/blog/a/${a.id}`)"
      >
        <h2 class="post-title display">{{ a.title }}</h2>
        <div class="post-meta">
          <span class="cat-tag">{{ categoryName(a.category_id) }}</span>
          <span class="dot" />
          <span class="mono">{{ new Date(a.created_at).toLocaleDateString() }}</span>
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
      </article>
      <div v-if="!filtered.length" class="empty">这一过滤条件下还没有内容。</div>
    </section>
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
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 24px;
  cursor: pointer;
  transition: all var(--transition);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.post-card:hover {
  border-color: var(--ink);
  transform: translateY(-2px);
}
.post-title {
  font-size: 24px;
  line-height: 1.2;
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
  font-size: 14px;
  line-height: 1.7;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.empty {
  grid-column: 1 / -1;
  text-align: center;
  padding: 80px 0;
  color: var(--ink-mute);
}
@media (max-width: 760px) {
  .hero-title { font-size: 36px; }
  .list { grid-template-columns: 1fr; }
}
</style>