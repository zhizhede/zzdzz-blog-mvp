import { http, type ApiResponse } from './http'

export interface AuthUser {
  id: number
  uuid: string
  username: string
  is_admin: boolean
  password_hint?: string
}

export interface LoginResult {
  token: string
  user: AuthUser
  expires_in: number
}

export const authApi = {
  login: (username: string, password: string) =>
    http.post<any, ApiResponse<LoginResult>>('/auth/login', { username, password }),
  // 开放注册: 用户名可重名, 注册成功即返回 token(注册即登录); password_hint 选填
  register: (username: string, password: string, passwordHint?: string) =>
    http.post<any, ApiResponse<LoginResult>>('/auth/register', {
      username,
      password,
      password_hint: passwordHint || '',
    }),
  // 忘记密码: 按用户名查注册时留的密码提示(公开接口, 未设置/用户不存在均返回空串)
  passwordHint: (username: string) =>
    http.post<any, ApiResponse<{ username: string; password_hint: string }>>('/auth/password-hint', {
      username,
    }),
  me: () => http.get<any, ApiResponse<AuthUser>>('/auth/me'),
  changeOwnPassword: (oldPassword: string, newPassword: string) =>
    http.put<any, ApiResponse<null>>('/auth/password', {
      old_password: oldPassword,
      new_password: newPassword,
    }),
  // 改密码提示: 需携带当前密码验证(提示是公开可查的, 改提示等于改找回入口)
  changeOwnPasswordHint: (password: string, passwordHint: string) =>
    http.put<any, ApiResponse<null>>('/auth/password-hint', {
      password,
      password_hint: passwordHint,
    }),
}

export * from './ai'
export { aiApi } from './ai'

export * from './writing'

export * from './user'
export { userApi } from './user'

export * from './tag'
export { tagApi } from './tag'

export * from './site'
export { siteApi } from './site'

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

export interface VisitLog {
  id: number
  ip: string
  user_id: number | null
  /** LEFT JOIN users 得到的用户名, 用户已删/不存在时为 null */
  username: string | null
  path: string
  user_agent: string
  created_at: string
}

export interface VisitLogListResult {
  total: number
  page: number
  size: number
  items: VisitLog[]
}

// 访问日志(0016): 仅 admin, 数据由后端全局中间件写入, 每位访客每天至多一条
export const visitLogApi = {
  list: (params: {
    page?: number
    page_size?: number
    ip?: string
    user?: string
    path?: string
    ua?: string
  }) => http.get<any, ApiResponse<VisitLogListResult>>('/visit-logs', { params }),
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