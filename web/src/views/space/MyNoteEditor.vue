<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { articleApi, categoryApi, type Category } from '../../api'

const route = useRoute()
const router = useRouter()

const id = Number(route.params.id) || 0
const isEdit = id > 0

const categories = ref<Category[]>([])
const form = ref({
  title: '',
  slug: '',
  summary: '',
  content: '',
  category_id: 0 as number,
  visibility: 'public' as 'public' | 'private' | 'draft',
})
const saving = ref(false)

const fetchCategories = async () => {
  const res = await categoryApi.list()
  categories.value = res.data
}

const fetchArticle = async () => {
  if (!isEdit) return
  const res = await articleApi.get(id)
  form.value = {
    title: res.data.title,
    slug: res.data.slug,
    summary: res.data.summary,
    content: res.data.content,
    category_id: res.data.category_id,
    visibility: res.data.visibility || 'public',
  }
}

const handleSave = async () => {
  if (!form.value.title || !form.value.content || !form.value.category_id) {
    ElMessage.warning('请填写标题、正文、分类')
    return
  }
  saving.value = true
  try {
    if (isEdit) {
      await articleApi.update(id, form.value)
      ElMessage.success('已保存')
    } else {
      const res = await articleApi.create(form.value)
      ElMessage.success('已发布')
      router.replace(`/space/notes/${res.data.id}/edit`)
    }
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await fetchCategories()
  await fetchArticle()
})
</script>

<template>
  <div class="page-container">
    <div class="toolbar">
      <h2>{{ isEdit ? '编辑笔记' : '新笔记' }}</h2>
      <div>
        <el-button @click="router.push('/space/notes')">返回</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ isEdit ? '保存' : '发布' }}
        </el-button>
      </div>
    </div>

    <el-card>
      <el-form label-width="80px">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="笔记标题" maxlength="255" />
        </el-form-item>
        <el-form-item label="Slug">
          <el-input v-model="form.slug" placeholder="URL 友好标识,可空" />
        </el-form-item>
        <el-form-item label="摘要">
          <el-input v-model="form.summary" placeholder="一句话摘要" maxlength="500" />
        </el-form-item>
        <el-form-item label="分类" required>
          <el-select v-model="form.category_id" placeholder="选择分类" style="width: 200px">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="可见性">
          <el-radio-group v-model="form.visibility">
            <el-radio-button value="public">公开</el-radio-button>
            <el-radio-button value="private">仅自己</el-radio-button>
            <el-radio-button value="draft">草稿</el-radio-button>
          </el-radio-group>
          <span class="visibility-hint">
            {{ form.visibility === 'public' ? '所有人都能看到' : form.visibility === 'private' ? '仅你(凭 token)能看到,游客 404' : '草稿,连你自己访问 URL 都 404' }}
          </span>
        </el-form-item>
        <el-form-item label="正文" required>
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="18"
            placeholder="支持 Markdown"
            resize="vertical"
          />
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.toolbar h2 { margin: 0; }
.visibility-hint { margin-left: 12px; color: #909399; font-size: 13px; }
</style>