<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Delete, Picture } from '@element-plus/icons-vue'
import { siteApi, type IconMeta } from '../../api/site'
import IssueTag from '../../components/IssueTag.vue'

const meta = ref<IconMeta | null>(null)
const uploading = ref(false)
// 预览地址带时间戳, 上传/重置后换参数强刷自己的浏览器缓存
const previewTick = ref(Date.now())

const previewSrc = () =>
  meta.value?.custom
    ? `/favicon-128.png?v=${meta.value.updated_at}&t=${previewTick.value}`
    : `/favicon-128.png?t=${previewTick.value}`

const fetchMeta = async () => {
  const res = await siteApi.getIconMeta()
  meta.value = res.data
}

const onPickFile = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // 允许重复选择同一文件
  if (!file) return
  uploading.value = true
  try {
    const res = await siteApi.uploadIcon(file)
    if (meta.value) meta.value = { custom: true, updated_at: String(res.data.updated_at) }
    else meta.value = { custom: true, updated_at: String(res.data.updated_at) }
    previewTick.value = Date.now()
    ElMessage.success('图标已更新, 全站访客刷新即见新图')
  } finally {
    uploading.value = false
  }
}

const handleReset = async () => {
  await ElMessageBox.confirm('恢复为内置默认图标?', '提示', { type: 'warning' })
  await siteApi.resetIcon()
  meta.value = { custom: false, updated_at: '' }
  previewTick.value = Date.now()
  ElMessage.success('已恢复默认图标')
}

onMounted(fetchMeta)
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="ADMIN" text="SETTINGS" suffix="SITE" />
      <h1 class="display title">站点设置</h1>
    </div>

    <section class="card">
      <div class="card-head">
        <span class="mono label">FAVICON · 网站图标</span>
        <span :class="['mono', 'state', meta?.custom ? 'state-custom' : '']">
          {{ meta?.custom ? '自定义' : '内置默认' }}
        </span>
      </div>

      <div class="card-body">
        <div class="preview-box">
          <img :src="previewSrc()" alt="当前图标" class="preview-img" />
          <el-icon class="preview-fallback"><Picture /></el-icon>
        </div>

        <div class="ops">
          <p class="desc">
            上传一张方形图片(PNG/JPG, ≤5MB), 系统自动生成全套尺寸
            (SVG / ICO / 32~512 PNG / apple-touch-icon)。
            生成后 favicon 地址带新版本号, 所有访客无需清缓存即可看到新图标。
          </p>
          <div class="btn-row">
            <label :class="['upload-btn', uploading && 'disabled']">
              <el-icon><Upload /></el-icon>
              {{ uploading ? '生成中…' : '上传新图标' }}
              <input type="file" accept=".png,.jpg,.jpeg,image/png,image/jpeg" :disabled="uploading" @change="onPickFile" />
            </label>
            <button v-if="meta?.custom" class="reset-btn" @click="handleReset">
              <el-icon><Delete /></el-icon> 恢复默认
            </button>
          </div>
          <p v-if="meta?.custom && meta.updated_at" class="mono updated">
            上次更新: {{ new Date(Number(meta.updated_at) * 1000).toLocaleString() }}
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 24px; padding-bottom: 64px; }
.page-head { display: flex; flex-direction: column; gap: 10px; }
.title { font-size: 32px; line-height: 1.1; margin: 0; letter-spacing: -0.6px; }

.card {
  background: var(--bg-elev);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  max-width: 640px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid var(--rule-soft);
  background: var(--bg-sunken);
  border-radius: var(--radius) var(--radius) 0 0;
}
.label { color: var(--ink-mute); font-size: 11px; text-transform: uppercase; letter-spacing: 0.16em; }
.state { font-size: 11px; color: var(--ink-mute); }
.state-custom { color: var(--accent); }

.card-body { display: flex; gap: 24px; padding: 24px 20px; align-items: flex-start; }
.preview-box {
  position: relative;
  width: 96px;
  height: 96px;
  flex: none;
  border: 1px dashed var(--rule);
  border-radius: var(--radius);
  background:
    conic-gradient(#e8e8e8 0 25%, #f7f7f7 0 50%, #e8e8e8 0 75%, #f7f7f7 0) 0 0/16px 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.preview-img { width: 72px; height: 72px; object-fit: contain; }
.preview-fallback { position: absolute; color: var(--ink-faint); }

.ops { display: flex; flex-direction: column; gap: 14px; min-width: 0; }
.desc { margin: 0; color: var(--ink-soft); font-size: 13px; line-height: 1.8; }
.btn-row { display: flex; gap: 12px; flex-wrap: wrap; }
.upload-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--ink);
  color: var(--ink-on-inverse);
  padding: 9px 18px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
  transition: background var(--transition);
}
.upload-btn:hover { background: var(--accent); }
.upload-btn.disabled { opacity: 0.6; pointer-events: none; }
.upload-btn input { display: none; }
.reset-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid var(--rule-soft);
  color: var(--ink-soft);
  padding: 9px 16px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 12px;
  cursor: pointer;
  transition: all var(--transition);
}
.reset-btn:hover { color: var(--danger); border-color: var(--danger); }
.updated { margin: 0; color: var(--ink-mute); font-size: 12px; }
</style>
