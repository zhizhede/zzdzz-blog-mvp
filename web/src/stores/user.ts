import { defineStore } from 'pinia'
import { authApi } from '../api'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userId: Number(localStorage.getItem('userId') || 0),
    username: localStorage.getItem('username') || '',
    isAdmin: localStorage.getItem('isAdmin') === '1',
  }),
  actions: {
    setAuth(token: string, userId: number, username: string, isAdmin: boolean) {
      this.token = token
      this.userId = userId
      this.username = username
      this.isAdmin = isAdmin
      localStorage.setItem('token', token)
      localStorage.setItem('userId', String(userId))
      localStorage.setItem('username', username)
      localStorage.setItem('isAdmin', isAdmin ? '1' : '0')
    },
    logout() {
      this.token = ''
      this.userId = 0
      this.username = ''
      this.isAdmin = false
      localStorage.removeItem('token')
      localStorage.removeItem('userId')
      localStorage.removeItem('username')
      localStorage.removeItem('isAdmin')
    },
    // 页面刷新后从 /auth/me 拉一次最新状态, 修复 localStorage 缺失字段导致的 isAdmin=false
    async refresh() {
      if (!this.token) return
      try {
        const res = await authApi.me()
        this.userId = res.data.id
        this.username = res.data.username
        this.isAdmin = !!res.data.is_admin
        localStorage.setItem('userId', String(res.data.id))
        localStorage.setItem('username', res.data.username)
        localStorage.setItem('isAdmin', res.data.is_admin ? '1' : '0')
      } catch {
        // token 失效 / 网络问题 — 让 http 拦截器统一处理(401 会清掉 token)
      }
    },
  },
})