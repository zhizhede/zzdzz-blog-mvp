<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { articleApi, categoryApi, tagApi, type Category, type Tag } from '../../api'

const route = useRoute()
const router = useRouter()

const id = Number(route.params.id) || 0
const isEdit = id > 0

const categories = ref<Category[]>([])
const allTags = ref<Tag[]>([])
const form = ref({
  title: '',
  slug: '',
  summary: '',
  content: '',
  category_id: 0 as number,
  tag_ids: [] as number[],
  visibility: 'public' as 'public' | 'private' | 'draft',
})
const saving = ref(false)
const lastSavedAt = ref<string | null>(null)
const saveError = ref<string | null>(null)
const statusText = computed(() => {
  if (saveError.value) return `未同步: ${saveError.value}`
  if (saving.value) return '保存中…'
  if (lastSavedAt.value) return `已自动保存 ${new Date(lastSavedAt.value).toLocaleTimeString()}`
  return '尚未自动保存'
})

const fetchCategories = async () => {
  const res = await categoryApi.list()
  categories.value = res.data
}
const fetchTags = async () => {
  const res = await tagApi.list()
  allTags.value = res.data
}
const fetchArticle = async () => {
  if (!isEdit) return
  const res = await articleApi.getWithTags(id)
  form.value = {
    title: res.data.title,
    slug: res.data.slug,
    summary: res.data.summary,
    content: res.data.content,
    category_id: res.data.category_id,
    tag_ids: (res.data.tags || []).map((t) => t.id),
    visibility: res.data.visibility || 'public',
  }
  lastSavedAt.value = res.data.last_autosaved_at
}

const handleSave = async () => {
  if (!form.value.title || !form.value.content || !form.value.category_id) {
    ElMessage.warning('请填写标题、正文、分类')
    return
  }
  saving.value = true
  try {
    const payload = { ...form.value, tag_ids: form.value.tag_ids }
    if (isEdit) {
      await articleApi.update(id, payload as any)
      ElMessage.success('已保存')
    } else {
      const res = await articleApi.create(payload as any)
      ElMessage.success('已发布')
      router.replace(`/admin/articles/${res.data.id}/edit`)
    }
  } finally {
    saving.value = false
  }
}

// ---------- autosave (v0.2 步骤 B: 只加 setTimeout + 调度, 不加 hook) ----------
const draftId = ref<number | null>(isEdit ? id : null)
const dirty = ref(false)
void dirty
let timer: number | null = null

// 步骤 C: localStorage 兜底
const LS_KEY_PREFIX = 'autosave:'
function lsKey(aid: number | null) { return LS_KEY_PREFIX + (aid ?? 'new') }
function writeLocalDraft() {
  try {
    localStorage.setItem(lsKey(draftId.value), JSON.stringify({
      title: form.value.title,
      summary: form.value.summary,
      content: form.value.content,
      category_id: form.value.category_id,
      tag_ids: form.value.tag_ids,
      visibility: form.value.visibility,
      savedAt: new Date().toISOString(),
    }))
  } catch {}
}
void writeLocalDraft

function scheduleAutosave() {
  if (timer) window.clearTimeout(timer)
  timer = window.setTimeout(doAutosave, 8000)
}

function markDirty() {
  dirty.value = true
  saveError.value = null
  writeLocalDraft()
}

function onInput() {
  markDirty()
  scheduleAutosave()
}

async function doAutosave() {
  if (!form.value.title.trim() || !form.value.content.trim()) return
  saving.value = true
  try {
    if (!draftId.value) {
      const res = await articleApi.create({
        title: form.value.title,
        slug: form.value.slug,
        summary: form.value.summary,
        content: form.value.content,
        category_id: form.value.category_id || categories.value[0]?.id || 0,
        visibility: 'draft',
        tag_ids: form.value.tag_ids,
      })
      draftId.value = res.data.id
      window.history.replaceState({}, '', `/admin/articles/${res.data.id}/edit`)
      lastSavedAt.value = res.data.last_autosaved_at
    } else {
      const res = await articleApi.autosave(draftId.value, {
        title: form.value.title,
        summary: form.value.summary,
        content: form.value.content,
        category_id: form.value.category_id,
      })
      lastSavedAt.value = res.data.last_autosaved_at
    }
    saveError.value = null
  } catch (e: any) {
    saveError.value = e?.message || 'autosave failed'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await Promise.all([fetchCategories(), fetchTags()])
  await fetchArticle()
})

// 步骤 D: onBeforeUnmount
onBeforeUnmount(() => {
  if (timer) window.clearTimeout(timer)
})
</script>

<template>
  <div class="page-container">
    <div class="toolbar">
      <h2>{{ isEdit ? '编辑文章' : '新文章' }}</h2>
      <div>
        <el-button @click="router.push('/admin/articles')">返回</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ isEdit ? '保存' : '发布' }}
        </el-button>
      </div>
    </div>

    <p :class="['status', saveError && 'error']">{{ statusText }}</p>

    <el-card>
      <el-form label-width="80px">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="文章标题" maxlength="255" @input="onInput" />
        </el-form-item>
        <el-form-item label="Slug">
          <el-input v-model="form.slug" placeholder="URL 友好标识,可空" />
        </el-form-item>
        <el-form-item label="摘要">
          <el-input v-model="form.summary" placeholder="一句话摘要" maxlength="500" @input="onInput" />
        </el-form-item>
        <el-form-item label="分类" required>
          <el-select v-model="form.category_id" placeholder="选择分类" style="width: 200px" @change="onInput">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签">
          <el-select
            v-model="form.tag_ids"
            multiple
            collapse-tags
            placeholder="选择标签(可多选)"
            style="width: 100%"
            @change="onInput"
          >
            <el-option v-for="t in allTags" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="可见性">
          <el-radio-group v-model="form.visibility" @change="onInput">
            <el-radio-button value="public">公开</el-radio-button>
            <el-radio-button value="private">仅自己</el-radio-button>
            <el-radio-button value="draft">草稿</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="正文" required>
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="18"
            placeholder="支持 Markdown"
            resize="vertical"
            @input="onInput"
          />
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.toolbar h2 { margin: 0; }
.status { color: #909399; font-size: 13px; margin: 0 0 12px; }
.status.error { color: #c45656; }
</style>
