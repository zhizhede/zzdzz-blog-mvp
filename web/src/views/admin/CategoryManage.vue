<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { categoryApi, articleApi, type Category } from '../../api'

const list = ref<Category[]>([])
const counts = ref<Record<number, number>>({})
const dialogVisible = ref(false)
const editing = ref<Category | null>(null)
const form = ref({ name: '', slug: '' })

const fetchList = async () => {
  const res = await categoryApi.list()
  list.value = res.data
  for (const c of res.data) {
    const r = await articleApi.list({ page: 1, size: 1, category_id: c.id })
    counts.value[c.id] = r.data.total
  }
}

const openCreate = () => {
  editing.value = null
  form.value = { name: '', slug: '' }
  dialogVisible.value = true
}

const openEdit = (c: Category) => {
  editing.value = c
  form.value = { name: c.name, slug: c.slug }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入名称')
    return
  }
  try {
    if (editing.value) {
      await categoryApi.update(editing.value.id, form.value.name, form.value.slug)
      ElMessage.success('已更新')
    } else {
      await categoryApi.create(form.value.name, form.value.slug)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    fetchList()
  } catch (e) {}
}

const handleDelete = async (c: Category) => {
  await ElMessageBox.confirm(`确认删除分类「${c.name}」?`, '提示', { type: 'warning' })
  await categoryApi.remove(c.id)
  ElMessage.success('已删除')
  fetchList()
}

onMounted(fetchList)
</script>

<template>
  <div class="page-container">
    <div class="toolbar">
      <h2>分类</h2>
      <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>新建分类</el-button>
    </div>

    <el-card>
      <el-table :data="list" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="slug" label="Slug" />
        <el-table-column label="文章数" width="100">
          <template #default="{ row }">{{ counts[row.id] || 0 }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button text type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑分类' : '新建分类'" width="420px">
      <el-form label-width="60px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="64" />
        </el-form-item>
        <el-form-item label="Slug">
          <el-input v-model="form.slug" placeholder="可选" maxlength="64" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.toolbar h2 { margin: 0; }
</style>