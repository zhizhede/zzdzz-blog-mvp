import { http, type ApiResponse } from './http'

export interface LoginResult {
  token: string
  user: { id: number; username: string; is_admin: boolean }
  expires_in: number
}

export const authApi = {
  login: (username: string, password: string) =>
    http.post<any, ApiResponse<LoginResult>>('/auth/login', { username, password }),
  me: () => http.get<any, ApiResponse<{ id: number; username: string; is_admin: boolean }>>('/auth/me'),
  changeOwnPassword: (oldPassword: string, newPassword: string) =>
    http.put<any, ApiResponse<null>>('/auth/password', {
      old_password: oldPassword,
      new_password: newPassword,
    }),
}

export * from './ai'
export { aiApi } from './ai'

export * from './user'
export { userApi } from './user'

export * from './tag'
export { tagApi } from './tag'

export interface Category {
  id: number
  name: string
  slug: string
  created_at: string
}

export const categoryApi = {
  list: () => http.get<any, ApiResponse<Category[]>>('/categories'),
  create: (name: string, slug: string) =>
    http.post<any, ApiResponse<Category>>('/categories', { name, slug }),
  update: (id: number, name: string, slug: string) =>
    http.put<any, ApiResponse<Category>>(`/categories/${id}`, { name, slug }),
  remove: (id: number) => http.delete<any, ApiResponse<null>>(`/categories/${id}`),
}

export interface Tag {
  id: number
  name: string
  slug: string
}

export interface Article {
  id: number
  title: string
  slug: string
  summary: string
  content: string
  category_id: number
  view_count: number
  created_at: string
  updated_at: string
  visibility: 'public' | 'private' | 'draft'
  author_id: number | null
  last_autosaved_at: string | null
  tags?: Tag[]
  tag_ids?: number[] | null
}

export interface ArticleListResult {
  total: number
  page: number
  size: number
  items: Article[]
}

export interface AutosaveInput {
  title: string
  summary?: string
  content: string
  category_id?: number
}

export interface AutosaveResult {
  id: number
  last_autosaved_at: string
  server_received_at: string
}

export const articleApi = {
  list: (params: {
    page?: number
    size?: number
    category_id?: number
    q?: string
    author_id?: number
    tag_id?: number
  }) => http.get<any, ApiResponse<ArticleListResult>>('/articles', { params }),
  get: (id: number) => http.get<any, ApiResponse<Article>>(`/articles/${id}`),
  getWithTags: (id: number) => http.get<any, ApiResponse<Article>>(`/articles/${id}/full`),
  create: (
    data: Omit<
      Article,
      | 'id'
      | 'view_count'
      | 'created_at'
      | 'updated_at'
      | 'visibility'
      | 'author_id'
      | 'last_autosaved_at'
      | 'tags'
    > & { visibility?: Article['visibility']; tag_ids?: number[] },
  ) => http.post<any, ApiResponse<Article>>('/articles', data),
  update: (
    id: number,
    data: Omit<
      Article,
      | 'id'
      | 'view_count'
      | 'created_at'
      | 'updated_at'
      | 'visibility'
      | 'author_id'
      | 'last_autosaved_at'
      | 'tags'
    > & { visibility?: Article['visibility']; tag_ids?: number[] },
  ) => http.put<any, ApiResponse<Article>>(`/articles/${id}`, data),
  remove: (id: number) => http.delete<any, ApiResponse<null>>(`/articles/${id}`),
  setVisibility: (id: number, visibility: Article['visibility']) =>
    http.patch<any, ApiResponse<Article>>(`/articles/${id}/visibility`, { visibility }),
  autosave: (id: number, data: AutosaveInput) =>
    http.put<any, ApiResponse<AutosaveResult>>(`/articles/${id}/autosave`, data),
  listMyDrafts: () =>
    http.get<any, ApiResponse<ArticleListResult>>('/articles/autosave/drafts'),
}