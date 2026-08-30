<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { articleApi, categoryApi, type Article, type Category } from '../../api'

const route = useRoute()
const router = useRouter()
const article = ref<Article | null>(null)
const categoryName = ref('')

onMounted(async () => {
  const id = Number(route.params.id)
  const res = await articleApi.getWithTags(id)
  article.value = res.data
  const cats = await categoryApi.list()
  const c = cats.data.find((x: Category) => x.id === res.data.category_id)
  categoryName.value = c?.name || '未分类'
})
</script>

<template>
  <div v-if="article" class="blog-detail">
    <el-button text @click="router.push('/blog')">← 返回列表</el-button>
    <h1 class="title">{{ article.title }}</h1>
    <div class="meta">
      <el-tag size="small">{{ categoryName }}</el-tag>
      <span>{{ new Date(article.created_at).toLocaleString() }}</span>
      <span>· {{ article.view_count }} 阅读</span>
      <template v-if="article.tags && article.tags.length">
        <span class="dot">·</span>
        <router-link
          v-for="t in article.tags"
          :key="t.id"
          :to="`/blog?tag_id=${t.id}`"
          class="tag-chip"
        >#{{ t.name }}</router-link>
      </template>
    </div>
    <div v-if="article.summary" class="summary">{{ article.summary }}</div>
    <pre class="content">{{ article.content }}</pre>
  </div>
</template>

<style scoped>
.blog-detail { background: #fff; padding: 28px 32px; border-radius: 8px; border: 1px solid #ebeef5; }
.title { margin: 16px 0 8px; font-size: 28px; color: #2c3e50; }
.meta { display: flex; gap: 12px; align-items: center; color: #909399; font-size: 13px; margin-bottom: 16px; flex-wrap: wrap; }
.meta .dot { color: #c0c4cc; }
.tag-chip {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: #409eff;
  border: 1px solid #d9ecff;
  background: #ecf5ff;
  padding: 2px 8px;
  border-radius: 999px;
  text-decoration: none;
}
.tag-chip:hover { background: #409eff; color: #fff; border-color: #409eff; }
.summary {
  background: #f6f8fa; padding: 12px 16px; border-left: 3px solid #409eff;
  color: #606266; margin-bottom: 20px; border-radius: 4px;
}
.content {
  white-space: pre-wrap; word-break: break-word;
  font-family: -apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 16px; line-height: 1.9; color: #2c3e50;
}
</style>