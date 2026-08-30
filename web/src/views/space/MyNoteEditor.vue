<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { articleApi, categoryApi, tagApi, type Article, type Tag } from '../../api'
import IssueTag from '../../components/IssueTag.vue'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id) || 0
const isEdit = id > 0

const categories = ref<{ id: number; name: string }[]>([])
const allTags = ref<Tag[]>([])

const form = ref({
  title: '',
  slug: '',
  summary: '',
  content: '',
  category_id: 0 as number,
  tag_ids: [] as number[],
  visibility: 'private' as 'public' | 'private' | 'draft',
})
const visibilityOptions: Array<'public' | 'private' | 'draft'> = ['private', 'public', 'draft']

const draftId = ref<number | null>(isEdit ? id : null)
const dirty = ref(false)
const lastSavedAt = ref<string | null>(null)
const saving = ref(false)
const saveError = ref<string | null>(null)

const fetchCategories = async () => {
  const res = await categoryApi.list()
  categories.value = res.data ?? []
}
const fetchTags = async () => {
  const res = await tagApi.list()
  allTags.value = res.data ?? []
}

async function loadNote(articleId: number) {
  const res = await articleApi.getWithTags(articleId)
  const a: Article = res.data
  form.value = {
    title: a.title,
    slug: a.slug,
    summary: a.summary,
    content: a.content,
    category_id: a.category_id,
    tag_ids: (a.tags ?? []).map((t) => t.id),
    visibility: a.visibility,
  }
  lastSavedAt.value = a.last_autosaved_at
}

const LS_KEY_PREFIX = 'autosave-space:'
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
function clearLocalDraft() {
  try { localStorage.removeItem(lsKey(draftId.value)) } catch {}
}
function markDirty() { dirty.value = true; saveError.value = null; writeLocalDraft() }

let timer: number | null = null
function scheduleAutosave() {
  if (timer) window.clearTimeout(timer)
  timer = window.setTimeout(doAutosave, 8000)
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
      window.history.replaceState({}, '', `/space/notes/${res.data.id}/edit`)
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

onBeforeUnmount(() => {
  if (timer) window.clearTimeout(timer)
})

function onInput() { markDirty(); scheduleAutosave() }

async function handleSave() {
  if (!form.value.title.trim() || !form.value.content.trim() || !form.value.category_id) {
    ElMessage.warning('请填写标题、正文、分类')
    return
  }
  if (!draftId.value || dirty.value) await doAutosave()
  if (!draftId.value) {
    ElMessage.error('草稿保存失败')
    return
  }
  const res = await articleApi.update(draftId.value, {
    title: form.value.title,
    slug: form.value.slug,
    summary: form.value.summary,
    content: form.value.content,
    category_id: form.value.category_id,
    visibility: form.value.visibility,
    tag_ids: form.value.tag_ids,
  })
  clearLocalDraft()
  ElMessage.success('已保存')
  router.replace(`/space/notes/${res.data.id}/edit`)
}

onMounted(async () => {
  await Promise.all([fetchCategories(), fetchTags()])
  if (isEdit) await loadNote(id)
  else form.value.category_id = categories.value[0]?.id || 0
})

const statusText = computed(() => {
  if (saveError.value) return `未同步: ${saveError.value}`
  if (saving.value) return '保存中…'
  if (lastSavedAt.value) return `已自动保存 ${new Date(lastSavedAt.value).toLocaleTimeString()}`
  return '尚未自动保存'
})
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="SPACE" text="NOTE" :suffix="isEdit ? `EDIT #${id}` : 'NEW'" />
      <h1 class="display title">{{ isEdit ? '编辑笔记' : '写新笔记' }}</h1>
      <p :class="['status', saveError && 'error']">{{ statusText }}</p>
    </div>

    <div class="grid">
      <div class="col col-form">
        <label class="field">
          <span class="mono label">TITLE</span>
          <input
            v-model="form.title"
            class="input title-input"
            placeholder="笔记标题"
            @input="onInput"
          />
        </label>

        <div class="row-2">
          <label class="field">
            <span class="mono label">CATEGORY</span>
            <select v-model="form.category_id" class="input" @change="onInput">
              <option :value="0" disabled>选择分类</option>
              <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </label>
          <label class="field">
            <span class="mono label">SLUG</span>
            <input v-model="form.slug" class="input" placeholder="(可选)" />
          </label>
        </div>

        <label class="field">
          <span class="mono label">SUMMARY</span>
          <textarea v-model="form.summary" class="input" rows="2" @input="onInput" />
        </label>

        <div class="field">
          <span class="mono label">VISIBILITY</span>
          <div class="vis-row">
            <button
              v-for="v in visibilityOptions"
              :key="v"
              type="button"
              :class="['vis-btn', form.visibility === v && 'active']"
              @click="form.visibility = v; onInput()"
            >
              {{ v === 'public' ? '公开' : v === 'private' ? '仅自己' : '草稿' }}
            </button>
          </div>
        </div>

        <div class="field">
          <span class="mono label">TAGS</span>
          <div class="tag-row">
            <button
              v-for="t in allTags"
              :key="t.id"
              type="button"
              :class="['tag-btn', form.tag_ids.includes(t.id) && 'active']"
              @click="(form.tag_ids.includes(t.id) ? form.tag_ids.splice(form.tag_ids.indexOf(t.id), 1) : form.tag_ids.push(t.id)); onInput()"
            >
              #{{ t.name }}
            </button>
          </div>
        </div>

        <label class="field">
          <span class="mono label">CONTENT · MARKDOWN</span>
          <textarea
            v-model="form.content"
            class="input mono-area"
            rows="18"
            @input="onInput"
          />
        </label>

        <div class="actions">
          <button class="text-btn" @click="router.push('/space/notes')">返回笔记</button>
          <button class="primary-btn" @click="handleSave">保存</button>
        </div>
      </div>

      <aside class="col col-preview">
        <div class="preview-head mono">PREVIEW</div>
        <article class="preview-body">
          <h2 class="preview-title display">{{ form.title || '无标题' }}</h2>
          <p v-if="form.summary" class="preview-summary">{{ form.summary }}</p>
          <pre class="preview-md">{{ form.content }}</pre>
        </article>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 24px; padding-bottom: 64px; }
.page-head { display: flex; flex-direction: column; gap: 10px; }
.title { font-size: 32px; line-height: 1.1; margin: 0; letter-spacing: -0.6px; }
.status { color: var(--ink-mute); font-size: 12px; margin: 0; }
.status.error { color: var(--danger); }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
.col { display: flex; flex-direction: column; gap: 16px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.input {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 8px 0;
  font-family: var(--font-body);
  font-size: 14px;
  color: var(--ink);
  outline: none;
  width: 100%;
  box-sizing: border-box;
}
.input:focus { border-bottom-color: var(--ink); }
.title-input { font-size: 22px; font-family: var(--font-display); font-weight: 500; }
.mono-area { font-family: var(--font-mono); font-size: 13px; line-height: 1.7; }
.row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.vis-row { display: flex; gap: 6px; }
.vis-btn {
  background: transparent;
  border: 1px solid var(--rule-soft);
  padding: 6px 12px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--ink-soft);
  cursor: pointer;
}
.vis-btn:hover { color: var(--ink); border-color: var(--ink-mute); }
.vis-btn.active { background: var(--ink); color: var(--ink-on-inverse); border-color: var(--ink); }
.tag-row { display: flex; flex-wrap: wrap; gap: 6px; }
.tag-btn {
  background: transparent;
  border: 1px solid var(--rule-soft);
  padding: 4px 10px;
  border-radius: 999px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--ink-soft);
  cursor: pointer;
}
.tag-btn.active { background: var(--accent); color: var(--accent-ink); border-color: var(--accent); }
.actions { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }
.text-btn {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 6px 0;
  font-family: var(--font-body);
  color: var(--ink);
  cursor: pointer;
}
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 10px 20px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
}
.primary-btn:hover { background: var(--accent); }
.col-preview {
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  position: sticky;
  top: 92px;
  max-height: calc(100vh - 120px);
  overflow: auto;
}
.preview-head { padding: 12px 20px; border-bottom: 1px solid var(--rule-soft); color: var(--ink-mute); background: var(--bg-sunken); }
.preview-body { padding: 24px; }
.preview-title { font-size: 24px; line-height: 1.15; margin: 0 0 12px; }
.preview-summary { border-left: 2px solid var(--accent); padding-left: 12px; color: var(--ink-soft); font-style: italic; margin: 0 0 16px; }
.preview-md { white-space: pre-wrap; font-family: var(--font-mono); font-size: 13px; line-height: 1.7; color: var(--ink-soft); margin: 0; }
@media (max-width: 960px) {
  .grid { grid-template-columns: 1fr; }
  .col-preview { position: static; max-height: none; }
}
</style>