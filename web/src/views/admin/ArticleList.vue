<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { articleApi, categoryApi, type Article, type Category } from '../../api'

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
    list.value = res.data.items
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

const fetchCategories = async () => {
  const res = await categoryApi.list()
  categories.value = res.data
}

const categoryName = (id: number) =>
  categories.value.find((c) => c.id === id)?.name || '-'

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
  <div class="page-container">
    <div class="toolbar">
      <h2>文章</h2>
      <el-button type="primary" @click="router.push('/admin/articles/new')">
        <el-icon><Plus /></el-icon>写新文章
      </el-button>
    </div>

    <el-card>
      <div class="filters">
        <el-select v-model="query.category_id" placeholder="全部分类" clearable style="width:160px" @change="fetchList">
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
        <el-input
          v-model="query.q"
          placeholder="搜索标题/摘要"
          clearable
          style="width:240px"
          @keyup.enter="fetchList"
          @clear="fetchList"
        />
        <el-button @click="fetchList">搜索</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="title" label="标题" min-width="220" />
        <el-table-column label="分类" width="100">
          <template #default="{ row }">{{ categoryName(row.category_id) }}</template>
        </el-table-column>
        <el-table-column prop="view_count" label="阅读" width="80" />
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="router.push(`/admin/articles/${row.id}/edit`)">编辑</el-button>
            <el-button text type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.size"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        style="margin-top: 16px"
        @current-change="fetchList"
        @size-change="fetchList"
      />
    </el-card>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.toolbar h2 { margin: 0; }
.filters { display: flex; gap: 12px; margin-bottom: 12px; }
</style>