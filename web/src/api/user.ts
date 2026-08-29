import { http, type ApiResponse } from './http'

export interface User {
  id: number
  username: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export const userApi = {
  list: () => http.get<any, ApiResponse<User[]>>('/users'),
  create: (username: string, password: string) =>
    http.post<any, ApiResponse<User>>('/users', { username, password }),
  changePassword: (id: number, oldPassword: string, newPassword: string) =>
    http.put<any, ApiResponse<null>>(`/users/${id}/password`, {
      old_password: oldPassword,
      new_password: newPassword,
    }),
  setActive: (id: number, isActive: boolean) =>
    http.patch<any, ApiResponse<null>>(`/users/${id}/active`, { is_active: isActive }),
}