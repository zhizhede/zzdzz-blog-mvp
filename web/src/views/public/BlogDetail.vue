<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { articleApi, categoryApi, type Article, type Category } from '../../api'
import IssueTag from '../../components/IssueTag.vue'
import Markdown from '../../components/Markdown.vue'
import AppFooter from '../../components/AppFooter.vue'

const route = useRoute()
const router = useRouter()
const article = ref<Article | null>(null)
const categoryName = ref('')
const loading = ref(true)
const notFound = ref(false)

onMounted(async () => {
  const id = Number(route.params.id)
  try {
    const res = await articleApi.getWithTags(id)
    article.value = res.data
    document.title = `${res.data.title} · zzdzz blog`
    const cats = await categoryApi.list()
    const c = cats.data?.find?.((x: Category) => x.id === res.data.category_id)
    categoryName.value = c?.name || '未分类'
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
})

const dateStr = computed(() => {
  if (!article.value) return ''
  return new Date(article.value.created_at).toLocaleString()
})
</script>

<template>
  <div class="container-narrow detail-page">
    <button class="back" @click="router.back()">← 返回列表</button>

    <div v-if="loading" class="state">加载中…</div>
    <div v-else-if="notFound || !article" class="state">
      <p class="state-title">文章不存在,或尚未公开。</p>
      <router-link to="/blog" class="state-link">去首页看看 →</router-link>
    </div>

    <template v-else>
      <IssueTag prefix="ESSAY" text="" :suffix="dateStr" />

    <h1 class="display title">{{ article.title }}</h1>

    <div class="meta">
      <span class="cat">{{ categoryName }}</span>
      <span class="dot" />
      <span class="mono">{{ dateStr }}</span>
      <span class="dot" />
      <span class="mono">{{ article.view_count }} 阅读</span>
      <template v-if="article.tags && article.tags.length">
        <span class="dot" />
        <router-link
          v-for="t in article.tags"
          :key="t.id"
          :to="`/blog?tag_id=${t.id}`"
          class="tag-chip"
        >#{{ t.name }}</router-link>
      </template>
    </div>

    <p v-if="article.summary" class="summary">{{ article.summary }}</p>

    <Markdown :source="article.content" />

    <AppFooter />
    </template>
  </div>
</template>

<style scoped>
.detail-page {
  padding: 40px 0 80px;
  display: flex;
  flex-direction: column;
}
.back {
  background: transparent;
  border: 0;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--ink-mute);
  cursor: pointer;
  padding: 0;
  margin-bottom: 32px;
  text-align: left;
  align-self: flex-start;
}
.back:hover { color: var(--accent); }
.state {
  padding: 80px 0;
  text-align: center;
  color: var(--ink-mute);
}
.state-title { margin: 0 0 16px; font-size: 16px; }
.state-link {
  color: var(--accent);
  font-family: var(--font-mono);
  font-size: 13px;
  text-decoration: none;
}
.title {
  font-size: 48px;
  line-height: 1.05;
  margin: 16px 0 8px;
  letter-spacing: -1.2px;
  color: var(--ink);
}
.meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  color: var(--ink-mute);
  font-size: 13px;
  margin-bottom: 16px;
}
.meta .cat {
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-size: 11px;
  color: var(--accent);
}
.meta .mono { font-family: var(--font-mono); }
.meta .dot {
  width: 3px;
  height: 3px;
  background: var(--ink-faint);
  border-radius: 50%;
}
.tag-chip {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--ink-mute);
  border: 1px solid var(--rule-soft);
  padding: 2px 8px;
  border-radius: 999px;
  text-decoration: none;
  transition: all var(--transition);
}
.tag-chip:hover {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border-color: var(--ink);
}
.summary {
  background: var(--bg-sunken);
  padding: 12px 16px;
  border-left: 3px solid var(--accent);
  color: var(--ink-soft);
  font-style: italic;
  margin: 0 0 24px;
  border-radius: var(--radius);
}
@media (max-width: 760px) {
  .title { font-size: 32px; }
}
</style>