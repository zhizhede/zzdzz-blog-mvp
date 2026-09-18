<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { visitLogApi, type VisitLog } from '../../api'
import IssueTag from '../../components/IssueTag.vue'

const list = ref<VisitLog[]>([])
const total = ref(0)
const query = ref({ page: 1, size: 20, ip: '', user: '', path: '', ua: '' })
const ipInput = ref('')
const userInput = ref('')
const pathInput = ref('')
const uaInput = ref('')

const fetchList = async () => {
  const res = await visitLogApi.list({
    page: query.value.page,
    page_size: query.value.size,
    ip: query.value.ip || undefined,
    user: query.value.user || undefined,
    path: query.value.path || undefined,
    ua: query.value.ua || undefined,
  })
  list.value = res.data?.items ?? []
  total.value = res.data?.total ?? 0
}

const applyFilter = () => {
  query.value.ip = ipInput.value.trim()
  query.value.user = userInput.value.trim()
  query.value.path = pathInput.value.trim()
  query.value.ua = uaInput.value.trim()
  query.value.page = 1
  fetchList()
}

const resetFilter = () => {
  ipInput.value = ''
  userInput.value = ''
  pathInput.value = ''
  uaInput.value = ''
  query.value.ip = ''
  query.value.user = ''
  query.value.path = ''
  query.value.ua = ''
  query.value.page = 1
  fetchList()
}

// 登录访客显示 "#ID 用户名", 匿名访客显示 "匿名"(虚线徽标), 天然区分用户名叫"匿名"的注册用户
const userLabel = (v: VisitLog) =>
  v.user_id ? `#${v.user_id}${v.username ? ' ' + v.username : ''}` : '匿名'

onMounted(fetchList)

const fmtTime = (iso: string) => new Date(iso).toLocaleString('zh-CN', { hour12: false })
</script>

<template>
  <div class="page">
    <div class="page-head">
      <IssueTag prefix="ADMIN" text="VISITS" :suffix="`${total} 条`" />
      <div class="head-row">
        <h1 class="display title">访问记录</h1>
        <button class="primary-btn" @click="fetchList">刷新</button>
      </div>
      <p class="hint">每位访客(IP)每天至多记录一条。用户筛选: 输入「匿名」查匿名访客(含用户名叫"匿名"的注册用户, 结果中 #ID 徽标者为注册用户); 输入 #数字按用户 ID 精确查; 输入其他文本按用户名模糊匹配。</p>
    </div>

    <div class="filter-row">
      <label class="filter-item">
        <span class="mono filter-label">IP · 精确</span>
        <input
          v-model="ipInput"
          class="input filter-input"
          placeholder="如 8.8.8.8"
          @keyup.enter="applyFilter"
        />
      </label>
      <label class="filter-item">
        <span class="mono filter-label">用户</span>
        <input
          v-model="userInput"
          class="input filter-input"
          placeholder="匿名 / #ID / 用户名"
          @keyup.enter="applyFilter"
        />
      </label>
      <label class="filter-item filter-wide">
        <span class="mono filter-label">路径 · 包含</span>
        <input
          v-model="pathInput"
          class="input filter-input"
          placeholder="如 /blog、wp-login"
          @keyup.enter="applyFilter"
        />
      </label>
      <label class="filter-item filter-wide">
        <span class="mono filter-label">UA · 包含</span>
        <input
          v-model="uaInput"
          class="input filter-input"
          placeholder="如 zgrab、iPhone"
          @keyup.enter="applyFilter"
        />
      </label>
      <button class="text-btn" @click="applyFilter">查询</button>
      <button class="text-btn" @click="resetFilter">重置</button>
    </div>

    <div class="visit-list">
      <div class="visit-row row-head mono">
        <span>时间</span>
        <span>IP</span>
        <span>用户</span>
        <span>路径</span>
        <span class="ua-col">USER AGENT</span>
      </div>
      <article v-for="v in list" :key="v.id" class="visit-row">
        <span class="mono time">{{ fmtTime(v.created_at) }}</span>
        <span class="mono ip">{{ v.ip }}</span>
        <span :class="['vis-pill', v.user_id ? 'vis-public' : 'vis-draft']" :title="userLabel(v)">
          {{ userLabel(v) }}
        </span>
        <span class="mono path" :title="v.path">{{ v.path }}</span>
        <span class="ua" :title="v.user_agent">{{ v.user_agent || '—' }}</span>
      </article>
      <div v-if="!list.length" class="empty">暂无访问记录。</div>
    </div>

    <div class="pager">
      <button class="text-btn" :disabled="query.page <= 1" @click="query.page--; fetchList()">← 上一页</button>
      <span class="mono pager-info">第 {{ query.page }} 页 · 共 {{ total }} 条</span>
      <button class="text-btn" :disabled="query.page * query.size >= total" @click="query.page++; fetchList()">下一页 →</button>
    </div>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 24px; padding-bottom: 64px; }
.page-head { display: flex; flex-direction: column; gap: 12px; }
.head-row { display: flex; justify-content: space-between; align-items: baseline; }
.title { font-size: 45px; line-height: 1; margin: 0; letter-spacing: -0.8px; }
.hint { color: var(--ink-mute); font-size: 15px; margin: 0; }
.primary-btn {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border: 0;
  padding: 10px 18px;
  border-radius: var(--radius);
  font-family: var(--font-mono);
  font-size: 15px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
}
.primary-btn:hover { background: var(--accent); }

.filter-row { display: flex; gap: 20px; align-items: flex-end; flex-wrap: wrap; }
.filter-item { display: flex; flex-direction: column; gap: 4px; }
.filter-item.filter-wide { flex: 1; min-width: 200px; }
.filter-label { color: var(--ink-mute); font-size: 13.75px; text-transform: uppercase; letter-spacing: 0.12em; }
.filter-input {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 8px 0;
  font-family: var(--font-mono);
  font-size: 16.25px;
  color: var(--ink);
  outline: none;
  width: 100%;
  box-sizing: border-box;
}
.filter-input:focus { border-bottom-color: var(--ink); }
.text-btn {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--rule-soft);
  padding: 4px 0;
  font-family: var(--font-body);
  font-size: 16.25px;
  color: var(--ink);
  cursor: pointer;
}
.text-btn:hover { color: var(--accent); border-bottom-color: var(--accent); }
.text-btn:disabled { color: var(--ink-faint); cursor: not-allowed; border-bottom-color: transparent; }

.visit-list { display: flex; flex-direction: column; }
.visit-row {
  display: grid;
  grid-template-columns: 170px 140px 150px minmax(0, 1fr) minmax(0, 1.1fr);
  gap: 16px;
  align-items: center;
  padding: 14px 12px;
  border-bottom: 1px solid var(--rule-soft);
}
.row-head {
  color: var(--ink-mute);
  font-size: 13.75px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  border-bottom-color: var(--rule);
}
.time { color: var(--ink-mute); font-size: 15px; }
.ip { font-size: 16.25px; color: var(--ink); }
.path {
  font-size: 15px;
  color: var(--ink-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ua {
  font-size: 15px;
  color: var(--ink-mute);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ua-col { text-align: left; }

.vis-pill {
  font-family: var(--font-mono);
  font-size: 13.75px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 3px 8px;
  border-radius: var(--radius);
  display: inline-block;
  width: fit-content;
}
.vis-public { background: var(--accent); color: var(--accent-ink); }
.vis-draft { background: transparent; color: var(--ink-mute); border: 1px dashed var(--rule-soft); }

.empty { padding: 60px 0; text-align: center; color: var(--ink-mute); }
.pager { display: flex; justify-content: space-between; align-items: center; padding-top: 16px; }
.pager-info { color: var(--ink-mute); font-size: 15px; }

@media (max-width: 1100px) {
  .visit-row { grid-template-columns: 150px 130px 130px minmax(0, 1fr); }
  .ua, .ua-col { display: none; }
}
</style>
