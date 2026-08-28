import { http, type ApiResponse } from './http'

export interface AIConversation {
  id: number
  user_id: number
  title: string
  created_at: string
  updated_at: string
}

export interface AIMessage {
  id: number
  conversation_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  created_at: string
}

export const aiApi = {
  listConversations: () =>
    http.get<any, ApiResponse<AIConversation[]>>('/ai/conversations'),
  createConversation: () =>
    http.post<any, ApiResponse<AIConversation>>('/ai/conversations'),
  renameConversation: (id: number, title: string) =>
    http.patch<any, ApiResponse<null>>(`/ai/conversations/${id}`, { title }),
  deleteConversation: (id: number) =>
    http.delete<any, ApiResponse<null>>(`/ai/conversations/${id}`),
  listMessages: (id: number, limit = 50) =>
    http.get<any, ApiResponse<AIMessage[]>>(
      `/ai/conversations/${id}/messages`,
      { params: { limit } },
    ),
}