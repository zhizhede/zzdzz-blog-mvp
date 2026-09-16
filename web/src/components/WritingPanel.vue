<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  composeStream,
  stripFences,
  writingApi,
  type ArticleVersion,
  type ComposeAction,
  type StyleProfile,
  type StyleSample,
} from '../api/writing'

// AI 写作面板(设计 doc/v0.4-tech-design.md §7):
// 语料 → ① 提大纲 → ② 成稿 → ③ 按指令调整(循环), 采用时整体交给父编辑器.
// 有 articleId 时: 采用前自动存快照(父级处理), 并提供版本列表/回滚(§6).
const props = defineProps<{ editorContent: string; articleId?: number | null }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'adopt', p: { content: string; action: ComposeAction; instruction: string }): void
  (e: 'restored', a: { title: string; summary: string; content: string }): void
}>()

const material = ref('')
const outline = ref('')
const instruction = ref('')
const result = ref('')
const busy = ref<ComposeAction | null>(null)
const errorMsg = ref('')

// ---- 风格卡(§5) ----
const profile = ref<StyleProfile | null>(null)
const profileDraft = ref('')
const deriving = ref(false)
const savingProfile = ref(false)
const useOwnStyle = ref(true)
const styleSamples = ref<StyleSample[]>([])
const styleMissing = ref(false) // 后端 meta 事件: 请求了风格卡但查不到

// ---- 版本快照(§6) ----
const versions = ref<ArticleVersion[]>([])
const versionsLoading = ref(false)
const versionBusy = ref<number | null>(null)

const originLabels: Record<string, string> = {
  manual: '手动备份',
  ai_outline: '采用大纲前',
  ai_draft: '采用成稿前',
  ai_refine: '采用调整前',
  pre_restore: '回滚前备份',
}

onMounted(async () => {
  await loadProfile()
  if (!profile.value && !deriving.value) await deriveProfile(true)
  if (props.articleId) await loadVersions()
})

async function loadProfile() {
  try {
    profile.value = (await writingApi.getStyleProfile()).data ?? null
    if (profile.value) profileDraft.value = profile.value.profile
  } catch {}
}

// 首次打开自动提炼; silent=true 时不弹成功提示
async function deriveProfile(silent = false) {
  deriving.value = true
  try {
    const res = await writingApi.deriveStyleProfile()
    profile.value = res.data.profile
    profileDraft.value = res.data.profile.profile
    styleSamples.value = res.data.samples ?? []
    styleMissing.value = false
    if (!silent) ElMessage.success('风格卡已更新')
  } catch {
    // 无文章 / 限频 / AI 未配置: 静默降级, 面板显示手写入口
  } finally {
    deriving.value = false
  }
}

async function reDerive() {
  if (profile.value?.source === 'manual') {
    if (!window.confirm('重新提炼会覆盖你手动修改过的风格卡,继续?')) return
  }
  await deriveProfile()
}

async function saveProfileManual() {
  if (!profileDraft.value.trim()) {
    ElMessage.warning('风格卡不能为空')
    return
  }
  savingProfile.value = true
  try {
    profile.value = (await writingApi.putStyleProfile(profileDraft.value)).data
    ElMessage.success('风格卡已保存')
  } catch {
  } finally {
    savingProfile.value = false
  }
}

async function loadVersions() {
  if (!props.articleId) return
  versionsLoading.value = true
  try {
    versions.value = (await writingApi.listVersions(props.articleId)).data ?? []
  } catch {
  } finally {
    versionsLoading.value = false
  }
}

// 回填: 取该版本内容到面板预览, 供继续调整或重新采用
async function backfillVersion(v: ArticleVersion) {
  if (!props.articleId || versionBusy.value) return
  versionBusy.value = v.id
  try {
    const full = (await writingApi.getVersion(props.articleId, v.id)).data
    if (result.value.trim()) pushHistory(resultActionOfResult.value ?? 'draft', result.value)
    result.value = full.content ?? ''
    trackAction('refine')
  } catch {
  } finally {
    versionBusy.value = null
  }
}

async function restoreVersion(v: ArticleVersion) {
  if (!props.articleId || versionBusy.value) return
  if (!window.confirm(`恢复到版本 #${v.id}?当前文章内容会先自动备份为「回滚前」版本。`)) return
  versionBusy.value = v.id
  try {
    const a = (await writingApi.restoreVersion(props.articleId, v.id)).data
    emit('restored', { title: a.title, summary: a.summary, content: a.content })
    ElMessage.success('文章已恢复到历史版本')
    await loadVersions()
  } catch {
  } finally {
    versionBusy.value = null
  }
}

// ---- 生成 ----

// 生成历史(内存, 最近 10 条): 兜底所有"被覆盖的结果"
interface HistoryItem { action: ComposeAction; content: string; at: string }
const history = ref<HistoryItem[]>([])
function pushHistory(action: ComposeAction, content: string) {
  if (!content.trim()) return
  history.value.unshift({ action, content, at: new Date().toLocaleTimeString() })
  if (history.value.length > 10) history.value.pop()
}

const resultLabel = computed(() => {
  if (busy.value === 'outline') return '大纲生成中…'
  if (busy.value) return '生成中…'
  if (result.value) return '结果预览'
  return ''
})

function restoreHistory(i: number) {
  const h = history.value[i]
  if (!h) return
  result.value = h.content
  if (h.action === 'outline') outline.value = h.content
}

// 当前 result 属于哪个动作(仅用于历史记录标注)
const resultActionOfResult = ref<ComposeAction | null>(null)
function trackAction(a: ComposeAction) {
  resultActionOfResult.value = a
}

async function run(action: ComposeAction) {
  if (busy.value) return
  if (action === 'outline' && !material.value.trim()) {
    ElMessage.warning('先粘贴原始语料')
    return
  }
  if (action === 'draft' && !material.value.trim()) {
    ElMessage.warning('成稿需要语料')
    return
  }
  if (action === 'refine' && !result.value.trim()) {
    ElMessage.warning('还没有可调整的稿子,先成稿或从编辑器取稿')
    return
  }
  if (action === 'refine' && !instruction.value.trim()) {
    ElMessage.warning('调整前先写一句指令')
    return
  }

  // 旧结果先入历史再清空; refine 的 draft 入参取自清空前的旧稿
  const prevResult = result.value
  if (prevResult.trim()) pushHistory(resultActionOfResult.value ?? 'draft', prevResult)
  result.value = ''
  trackAction(action)

  busy.value = action
  errorMsg.value = ''
  styleMissing.value = false
  if (!useOwnStyle.value) styleSamples.value = []
  let acc = ''
  try {
    await composeStream(
      {
        action,
        material: material.value,
        outline: action === 'draft' ? outline.value : undefined,
        draft: action === 'refine' ? prevResult : undefined,
        instruction: action === 'refine' ? instruction.value : undefined,
        use_own_style: useOwnStyle.value,
      },
      (chunk) => {
        acc += chunk
        result.value = acc
      },
      {
        onEvent: (obj) => {
          if (obj.sources) styleSamples.value = obj.sources
          if (obj.meta?.style_missing) styleMissing.value = true
        },
      },
    )
    result.value = stripFences(acc)
    if (action === 'outline') outline.value = result.value
  } catch (e: any) {
    errorMsg.value = e?.message || '生成失败'
    if (acc) result.value = stripFences(acc)
  } finally {
    busy.value = null
  }
}

function importFromEditor() {
  if (!props.editorContent.trim()) {
    ElMessage.warning('编辑器里还没有内容')
    return
  }
  pushHistory(resultActionOfResult.value ?? 'draft', result.value)
  result.value = props.editorContent
  trackAction('refine')
  ElMessage.success('已取编辑器内容,可写指令后调整')
}

function adopt() {
  if (!result.value.trim() || busy.value) return
  emit('adopt', {
    content: result.value,
    action: resultActionOfResult.value ?? 'draft',
    instruction: instruction.value,
  })
}
</script>

<template>
  <div class="wp-mask" @click.self="emit('close')">
    <aside class="wp-panel">
      <header class="wp-head">
        <span class="mono wp-title">AI WRITING</span>
        <button class="wp-x mono" @click="emit('close')">ESC</button>
      </header>

      <div class="wp-body">
        <details class="wp-style">
          <summary class="mono wp-label">
            风格卡
            <span v-if="profile" class="wp-style-meta">
              · {{ profile.source === 'manual' ? '手改' : '自动' }} ·
              {{ new Date(profile.updated_at).toLocaleDateString() }}
            </span>
            <span v-else-if="deriving" class="wp-style-meta"> · 正在学习你的文风…</span>
            <span v-else class="wp-style-meta"> · 未设置</span>
          </summary>
          <textarea
            v-model="profileDraft"
            class="wp-input"
            rows="5"
            :placeholder="deriving ? '正在从你的历史文章提炼文风…' : '没有自动提炼成功?可以手写:语气、句式、口头禅、结构套路…'"
          />
          <div class="wp-steps">
            <button class="wp-btn" :disabled="savingProfile || !profileDraft.trim()" @click="saveProfileManual">
              保存手改
            </button>
            <button class="wp-btn ghost" :disabled="deriving" @click="reDerive">
              {{ deriving ? '提炼中…' : '重新提炼' }}
            </button>
          </div>
          <label v-if="profile" class="wp-style-toggle">
            <input v-model="useOwnStyle" type="checkbox" />
            <span class="mono">生成时使用我的文风</span>
          </label>
          <p v-if="styleSamples.length" class="mono wp-samples">
            参考文章:{{ styleSamples.map((s) => `#${s.id}《${s.title}》`).join(' ') }}
          </p>
        </details>

        <p v-if="styleMissing" class="wp-banner">没找到你的风格卡,本次使用通用文风。可在上方「风格卡」里手写或重新提炼。</p>

        <label class="wp-field">
          <span class="mono wp-label">语料 · 原始想法 / 笔记 / 口述</span>
          <textarea
            v-model="material"
            class="wp-input"
            rows="5"
            placeholder="把想法、素材、草稿原文粘到这里,不需要整理,想到什么写什么"
          />
        </label>

        <div class="wp-steps">
          <button class="wp-btn" :disabled="!!busy || !material.trim()" @click="trackAction('outline'); run('outline')">
            {{ busy === 'outline' ? '大纲生成中…' : '① 提大纲' }}
          </button>
          <button class="wp-btn primary" :disabled="!!busy || !material.trim()" @click="trackAction('draft'); run('draft')">
            {{ busy === 'draft' ? '成稿中…' : '② 成稿' }}
          </button>
        </div>

        <label v-if="outline" class="wp-field">
          <span class="mono wp-label">大纲 · 可直接编辑</span>
          <textarea v-model="outline" class="wp-input" rows="4" />
        </label>

        <label class="wp-field">
          <span class="mono wp-label">调整指令</span>
          <input
            v-model="instruction"
            class="wp-input"
            placeholder="例: 第三节展开写,融入我补充的这段,语气更口语"
            @keyup.enter="trackAction('refine'); run('refine')"
          />
        </label>

        <div class="wp-steps">
          <button class="wp-btn" :disabled="!!busy || !result.trim() || !instruction.trim()" @click="trackAction('refine'); run('refine')">
            {{ busy === 'refine' ? '调整中…' : '③ 按指令调整' }}
          </button>
          <button class="wp-btn ghost" :disabled="!!busy" @click="importFromEditor">从编辑器取稿</button>
        </div>

        <div v-if="result || busy" class="wp-field">
          <span class="mono wp-label">{{ resultLabel }}</span>
          <pre class="wp-result">{{ result }}<span v-if="busy" class="wp-cursor">▌</span></pre>
        </div>

        <p v-if="errorMsg" class="wp-error">{{ errorMsg }}</p>

        <details v-if="history.length" class="wp-history">
          <summary class="mono wp-label">生成历史({{ history.length }}) · 点击回填</summary>
          <button v-for="(h, i) in history" :key="i" class="wp-hist-btn mono" @click="restoreHistory(i)">
            #{{ history.length - i }} {{ h.action }} · {{ h.at }}
          </button>
        </details>

        <details v-if="articleId" class="wp-history" @toggle="versions.length || versionsLoading || loadVersions()">
          <summary class="mono wp-label">
            文章版本({{ versions.length }}) · 采用/回滚前自动备份
          </summary>
          <div v-if="versionsLoading" class="mono wp-hist-btn">加载中…</div>
          <div v-for="v in versions" :key="v.id" class="wp-ver-row">
            <span class="mono wp-ver-label">
              #{{ v.id }} {{ originLabels[v.origin] || v.origin }} · {{ new Date(v.created_at).toLocaleString() }}
            </span>
            <span class="wp-ver-actions">
              <button class="wp-hist-btn mono" :disabled="!!versionBusy" @click="backfillVersion(v)">回填预览</button>
              <button class="wp-hist-btn mono wp-danger" :disabled="!!versionBusy" @click="restoreVersion(v)">恢复</button>
            </span>
          </div>
        </details>
      </div>

      <footer class="wp-foot">
        <span v-if="!articleId" class="mono wp-tip">新文章未保存,采用不产生服务端快照,可在「生成历史」找回旧结果</span>
        <span v-else class="mono wp-tip">采用前会自动把编辑器当前内容备份为版本</span>
        <button class="wp-btn primary" :disabled="!result.trim() || !!busy" @click="adopt">采用到编辑器</button>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.wp-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.28);
  z-index: 200;
  display: flex;
  justify-content: flex-end;
}
.wp-panel {
  width: min(460px, 92vw);
  height: 100vh;
  background: var(--bg-elev);
  border-left: 1px solid var(--rule-soft);
  display: flex;
  flex-direction: column;
  box-shadow: -12px 0 32px rgba(0, 0, 0, 0.12);
}
.wp-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px;
  border-bottom: 1px solid var(--rule-soft);
  background: var(--bg-sunken);
}
.wp-title { font-size: 13.75px; letter-spacing: 0.16em; color: var(--ink-mute); }
.wp-x {
  background: transparent;
  border: 0;
  color: var(--ink-mute);
  cursor: pointer;
  font-size: 13.75px;
}
.wp-x:hover { color: var(--ink); }
.mono { font-family: var(--font-mono); }

.wp-body {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.wp-field { display: flex; flex-direction: column; gap: 6px; }
.wp-label {
  font-size: 12.5px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--ink-mute);
}
.wp-input {
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 8px 10px;
  font-family: var(--font-body);
  font-size: 16.25px;
  line-height: 1.6;
  color: var(--ink);
  outline: none;
  resize: vertical;
  width: 100%;
  box-sizing: border-box;
}
.wp-input:focus { border-color: var(--ink-mute); }

.wp-style {
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.wp-style summary { cursor: pointer; }
.wp-style-meta { text-transform: none; letter-spacing: 0.04em; }
.wp-style-toggle { display: flex; align-items: center; gap: 6px; font-size: 15px; color: var(--ink-soft); }
.wp-samples { margin: 0; font-size: 12.5px; color: var(--ink-mute); line-height: 1.7; }

.wp-banner {
  margin: 0;
  border: 1px dashed var(--rule-soft);
  border-radius: var(--radius);
  padding: 8px 10px;
  font-size: 15px;
  color: var(--ink-soft);
  background: var(--bg-sunken);
}

.wp-steps { display: flex; gap: 8px; }
.wp-btn {
  flex: 1;
  background: transparent;
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 9px 12px;
  font-family: var(--font-mono);
  font-size: 15px;
  color: var(--ink-soft);
  cursor: pointer;
  transition: all var(--transition);
}
.wp-btn:hover:not(:disabled) { color: var(--ink); border-color: var(--ink-mute); }
.wp-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.wp-btn.primary { background: var(--ink); color: var(--ink-on-inverse); border-color: var(--ink); }
.wp-btn.primary:hover:not(:disabled) { background: var(--accent); border-color: var(--accent); color: var(--accent-ink); }
.wp-btn.ghost { border-style: dashed; }

.wp-result {
  margin: 0;
  background: var(--bg-sunken);
  border: 1px solid var(--rule-soft);
  border-radius: var(--radius);
  padding: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-mono);
  font-size: 15px;
  line-height: 1.7;
  color: var(--ink-soft);
  max-height: 40vh;
  overflow: auto;
}
.wp-cursor { animation: wp-blink 1s step-end infinite; color: var(--accent); }
@keyframes wp-blink { 50% { opacity: 0; } }

.wp-error {
  margin: 0;
  color: var(--danger);
  font-size: 15px;
  line-height: 1.6;
}

.wp-history { display: flex; flex-direction: column; gap: 6px; }
.wp-history summary { cursor: pointer; }
.wp-hist-btn {
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 6px 0;
  font-size: 13.75px;
  color: var(--ink-mute);
  cursor: pointer;
}
.wp-hist-btn:hover:not(:disabled) { color: var(--ink); }
.wp-hist-btn:disabled { opacity: 0.5; cursor: wait; }
.wp-danger:hover:not(:disabled) { color: var(--danger); }

.wp-ver-row { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.wp-ver-row .wp-hist-btn { border-bottom: 0; padding: 4px 0; }
.wp-ver-label { flex: 1; font-size: 12.5px; color: var(--ink-mute); }
.wp-ver-actions { display: flex; gap: 10px; }

.wp-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  border-top: 1px solid var(--rule-soft);
  background: var(--bg-sunken);
}
.wp-tip { flex: 1; font-size: 12.5px; color: var(--ink-mute); line-height: 1.5; }
.wp-foot .wp-btn { flex: none; padding: 9px 18px; }
</style>
