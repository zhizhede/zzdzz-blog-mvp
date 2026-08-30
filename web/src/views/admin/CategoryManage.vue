<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { categoryApi, articleApi, type Category } from '../../api'
import IssueTag from '../../components/IssueTag.vue'

const list = ref<Category[]>([])
const counts = ref<Record<number, number>>({})
const dialogVisible = ref(false)
const editing = ref<Category | null>(null)
const form = ref({ name: '', slug: '' })

const fetchList = async () => {
  const res = await categoryApi.list()
  list.value = res.data ?? []
  for (const c of list.value) {
    const r = await articleApi.list({ page: 1, size: 1, category_id: c.id })
    counts.value[c.id] = r.data.total ?? 0
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
  } catch {}
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
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="ADMIN" text="CATEGORIES" :suffix="`${list.length} 项`" />
      <div class="head-row">
        <h1 class="display title">分类</h1>
        <button class="primary-btn" @click="openCreate">
          <span class="mono">＋</span> 新建分类
        </button>
      </div>
    </div>

    <div class="grid">
      <article v-for="c in list" :key="c.id" class="cat-card">
        <div class="cat-meta">
          <span class="mono">#{{ c.id }}</span>
          <span class="mono count">{{ counts[c.id] ?? 0 }} 篇</span>
        </div>
        <h2 class="cat-name display">{{ c.name }}</h2>
        <p class="cat-slug mono">/{{ c.slug || c.id }}</p>
        <div class="cat-actions">
          <button class="text-btn" @click="openEdit(c)">编辑</button>
          <button class="text-btn danger" @click="handleDelete(c)">删除</button>
        </div>
      </article>
      <div v-if="!list.length" class="empty">还没有分类。</div>
    </div>

    <div v-if="dialogVisible" class="overlay" @click.self="dialogVisible = false">
      <div class="dialog">
        <p class="mono d-tag">{{ editing ? 'EDIT CATEGORY' : 'NEW CATEGORY' }}</p>
        <h2 class="display d-title">{{ editing ? '编辑分类' : '新建分类' }}</h2>
        <label class="field">
          <span class="mono label">NAME</span>
          <input v-model="form.name" class="input" maxlength="64" />
        </label>
        <label class="field">
          <span class="mono label">SLUG</span>
          <input v-model="form.slug" class="input" placeholder="(可选)" maxlength="64" />
        </label>
        <div class="d-row">
          <button class="text-btn" @click="dialogVisible = false">取消</button>
          <button class="primary-btn" @click="handleSubmit">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 24px; padding-bottom: 64px; }
.page-head { display: flex; flex-direction: column; gap: 12px; }
.head-row { display: flex; justify-content: space-between; align-items: baseline; }
.title { font-size: 36px; line-height: 1; margin: 0; letter-spacing: -0.8px; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 10px 18px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.primary-btn:hover { background: var(--accent); }
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.cat-card {
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: all var(--transition);
}
.cat-card:hover { border-color: var(--ink); transform: translateY(-2px); }
.cat-meta { display: flex; justify-content: space-between; color: var(--ink-mute); font-size: 11px; }
.cat-meta .count { color: var(--accent); }
.cat-name { font-size: 22px; margin: 0; }
.cat-slug { color: var(--ink-mute); font-size: 12px; margin: 0; }
.cat-actions { display: flex; gap: 12px; padding-top: 6px; border-top: 1px solid var(--rule-soft); margin-top: 4px; }
.text-btn {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 4px 0;
  font-family: var(--font-body);
  font-size: 13px;
  color: var(--ink);
  cursor: pointer;
}
.text-btn:hover { color: var(--accent); border-bottom-color: var(--accent); }
.text-btn.danger:hover { color: var(--danger); border-bottom-color: var(--danger); }
.empty { grid-column: 1 / -1; text-align: center; padding: 60px 0; color: var(--ink-mute); }

.overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
  z-index: 100;
}
.dialog {
  background: var(--bg);
  border: 1px solid var(--rule);
  border-radius: var(--radius);
  padding: 28px;
  width: 420px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.d-tag { color: var(--accent); margin: 0; }
.d-title { font-size: 24px; margin: 0; }
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
}
.input:focus { border-bottom-color: var(--ink); }
.d-row { display: flex; gap: 12px; justify-content: flex-end; padding-top: 8px; }
</style>