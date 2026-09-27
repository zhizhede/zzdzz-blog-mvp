import { http, type ApiResponse } from './http'

// 快捷用语板(0019): 用户自预设短语, 编辑器点击即插入正文光标处.
// 数据按用户隔离, 属主判定在服务端按 token 完成.
export interface QuickPhrase {
  id: number
  user_id: number
  content: string
  sort_order: number
  created_at: string
  updated_at: string
}

export const quickPhraseApi = {
  list: () => http.get<any, ApiResponse<QuickPhrase[]>>('/quick-phrases'),
  create: (content: string) =>
    http.post<any, ApiResponse<QuickPhrase>>('/quick-phrases', { content }),
  update: (id: number, content: string) =>
    http.put<any, ApiResponse<QuickPhrase>>(`/quick-phrases/${id}`, { content }),
  remove: (id: number) => http.delete<any, ApiResponse<null>>(`/quick-phrases/${id}`),
}
