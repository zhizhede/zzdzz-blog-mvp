import { http, type ApiResponse } from './http'
import { useUserStore } from '../stores/user'

export type ComposeAction = 'outline' | 'draft' | 'refine'

export interface ComposeReq {
  action: ComposeAction
  material?: string
  outline?: string
  draft?: string
  instruction?: string
  /** 请求注入风格卡; 后端查不到时降级通用文风并推 meta 事件 */
  use_own_style?: boolean
}

export interface StyleProfile {
  user_id: number
  profile: string
  source: 'auto' | 'manual'
  updated_at: string
}

export interface StyleSample {
  id: number
  title: string
}

export interface ArticleVersion {
  id: number
  article_id: number
  title: string
  summary?: string
  /** 列表接口不含 content, 单查接口才有 */
  content?: string
  origin: 'manual' | 'ai_outline' | 'ai_draft' | 'ai_refine' | 'pre_restore'
  note: string
  created_by: number
  created_at: string
}

// compose 走 fetch + ReadableStream 解析 SSE(与 AICenter.vue 同一套写法):
// EventSource 不支持 POST, 且需要带 Authorization 头.
export async function composeStream(
  req: ComposeReq,
  onDelta: (chunk: string) => void,
  opts: { signal?: AbortSignal; onEvent?: (obj: Record<string, any>) => void } = {},
): Promise<void> {
  const userStore = useUserStore()
  const resp = await fetch('/api/v1/ai/compose', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${userStore.token}`,
    },
    body: JSON.stringify(req),
    signal: opts.signal,
  })
  if (!resp.ok || !resp.body) {
    const errText = await resp.text().catch(() => '')
    throw new Error(errText || `请求失败 (${resp.status})`)
  }

  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n\n')
    buffer = lines.pop() || ''
    for (const line of lines) {
      const m = line.match(/^data:\s*(.*)$/)
      if (!m) continue
      const payload = m[1]
      if (payload === '[DONE]') return
      let obj: any
      try {
        obj = JSON.parse(payload)
      } catch {
        continue
      }
      if (obj.error) throw new Error(typeof obj.error === 'string' ? obj.error : JSON.stringify(obj.error))
      if (obj.delta) onDelta(obj.delta)
      else if (obj.sources || obj.meta) opts.onEvent?.(obj)
    }
  }
}

// 后端已对流式输出做首尾围栏剥壳, 这里对累计全文再兜底一次,
// 防止极端分片把围栏切成两段导致的漏网.
export function stripFences(text: string): string {
  let t = text.replace(/^\s*```[^\n]*\n/, '')
  t = t.replace(/\n?```[\s]*$/, '')
  return t
}

export const writingApi = {
  // ---- 风格卡 ----
  getStyleProfile: () => http.get<any, ApiResponse<StyleProfile | null>>('/writing/style-profile'),
  putStyleProfile: (profile: string) =>
    http.put<any, ApiResponse<StyleProfile>>('/writing/style-profile', { profile }),
  deriveStyleProfile: () =>
    http.post<any, ApiResponse<{ profile: StyleProfile; samples: StyleSample[] }>>(
      '/writing/style-profile/derive',
    ),
  // ---- 版本快照 ----
  listVersions: (articleId: number) =>
    http.get<any, ApiResponse<ArticleVersion[]>>(`/articles/${articleId}/versions`),
  getVersion: (articleId: number, vid: number) =>
    http.get<any, ApiResponse<ArticleVersion>>(`/articles/${articleId}/versions/${vid}`),
  createVersion: (articleId: number, origin: string, note = '') =>
    http.post<any, ApiResponse<ArticleVersion>>(`/articles/${articleId}/versions`, { origin, note }),
  restoreVersion: (articleId: number, vid: number) =>
    http.post<any, ApiResponse<{ id: number; title: string; summary: string; content: string }>>(
      `/articles/${articleId}/versions/${vid}/restore`,
    ),
}
