import { http, type ApiResponse } from './http'

export interface LoginResult {
  token: string
  user: { id: number; username: string }
  expires_in: number
}

export const authApi = {
  login: (username: string, password: string) =>
    http.post<any, ApiResponse<LoginResult>>('/auth/login', { username, password }),
  me: () => http.get<any, ApiResponse<{ id: number; username: string }>>('/auth/me'),
}

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
}

export interface ArticleListResult {
  total: number
  page: number
  size: number
  items: Article[]
}

export const articleApi = {
  list: (params: { page?: number; size?: number; category_id?: number; q?: string }) =>
    http.get<any, ApiResponse<ArticleListResult>>('/articles', { params }),
  get: (id: number) => http.get<any, ApiResponse<Article>>(`/articles/${id}`),
  create: (data: Omit<Article, 'id' | 'view_count' | 'created_at' | 'updated_at'>) =>
    http.post<any, ApiResponse<Article>>('/articles', data),
  update: (id: number, data: Omit<Article, 'id' | 'view_count' | 'created_at' | 'updated_at'>) =>
    http.put<any, ApiResponse<Article>>(`/articles/${id}`, data),
  remove: (id: number) => http.delete<any, ApiResponse<null>>(`/articles/${id}`),
}