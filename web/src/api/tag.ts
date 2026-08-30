import { http, type ApiResponse } from './http'

export interface Tag {
  id: number
  name: string
  slug: string
  created_at: string
}

export interface TagWithCount {
  id: number
  name: string
  slug: string
  created_at: string
  count: number
}

export const tagApi = {
  list: () => http.get<any, ApiResponse<TagWithCount[]>>('/tags'),
  create: (name: string, slug: string) =>
    http.post<any, ApiResponse<Tag>>('/tags', { name, slug }),
  update: (id: number, name: string, slug: string) =>
    http.put<any, ApiResponse<Tag>>(`/tags/${id}`, { name, slug }),
  remove: (id: number) =>
    http.delete<any, ApiResponse<null>>(`/tags/${id}`),
}
