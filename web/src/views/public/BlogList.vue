<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { articleApi, categoryApi, type Article, type Category } from '../../api'

const router = useRouter()
const articles = ref<Article[]>([])
const categories = ref<Category[]>([])
const filterCat = ref(0)

const fetchData = async () => {
  const [a, c] = await Promise.all([
    articleApi.list({ page: 1, size: 50, category_id: filterCat.value || undefined }),
    categoryApi.list(),
  ])
  articles.value = a.data.items
  categories.value = c.data
}

const filtered = computed(() =>
  filterCat.value ? articles.value.filter((a) => a.category_id === filterCat.value) : articles.value,
)

const categoryName = (id: number) => categories.value.find((c) => c.id === id)?.name || '未分类'
const summary = (a: Article) => a.summary || a.content.slice(0, 80).replace(/\n/g, ' ')

onMounted(fetchData)
</script>

<template>
  <div class="blog-list">
    <div class="filters">
      <span class="filter-label">分类:</span>
      <span :class="['tag', filterCat === 0 && 'active']" @click="filterCat = 0">全部</span>
      <span
        v-for="c in categories"
        :key="c.id"
        :class="['tag', filterCat === c.id && 'active']"
        @click="filterCat = c.id"
      >{{ c.name }}</span>
    </div>

    <article v-for="a in filtered" :key="a.id" class="post-card" @click="router.push(`/blog/a/${a.id}`)">
      <h2 class="title">{{ a.title }}</h2>
      <div class="meta">
        <el-tag size="small">{{ categoryName(a.category_id) }}</el-tag>
        <span>{{ new Date(a.created_at).toLocaleDateString() }}</span>
        <span>· {{ a.view_count }} 阅读</span>
      </div>
      <p class="excerpt">{{ summary(a) }}</p>
    </article>

    <el-empty v-if="!filtered.length" description="还没有文章,去后台写一篇吧" />
  </div>
</template>

<style scoped>
.blog-list { display: flex; flex-direction: column; gap: 20px; }
.filters { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 8px; }
.filter-label { color: #606266; font-size: 14px; }
.tag {
  padding: 4px 12px; border-radius: 16px; background: #f0f2f5;
  cursor: pointer; font-size: 13px; color: #606266;
}
.tag.active { background: #409eff; color: #fff; }
.post-card {
  background: #fff; border-radius: 8px; padding: 20px 24px;
  cursor: pointer; transition: all 0.2s;
  border: 1px solid #ebeef5;
}
.post-card:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.08); }
.title { margin: 0 0 8px; font-size: 20px; color: #2c3e50; }
.meta { display: flex; gap: 12px; align-items: center; color: #909399; font-size: 13px; margin-bottom: 8px; }
.excerpt { color: #606266; line-height: 1.6; margin: 0; }
</style>