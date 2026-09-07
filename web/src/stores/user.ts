import { defineStore } from 'pinia'
import { authApi } from '../api'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userId: Number(localStorage.getItem('userId') || 0),
    // userUuid 用户稳定唯一身份标识(0013 起); 0014 起用户名也全系统唯一, 登录/注册判定键为 username
    userUuid: localStorage.getItem('userUuid') || '',
    username: localStorage.getItem('username') || '',
    isAdmin: localStorage.getItem('isAdmin') === '1',
  }),
  actions: {
    setAuth(token: string, userId: number, username: string, isAdmin: boolean, userUuid = '') {
      this.token = token
      this.userId = userId
      this.userUuid = userUuid
      this.username = username
      this.isAdmin = isAdmin
      localStorage.setItem('token', token)
      localStorage.setItem('userId', String(userId))
      localStorage.setItem('userUuid', userUuid)
      localStorage.setItem('username', username)
      localStorage.setItem('isAdmin', isAdmin ? '1' : '0')
    },
    logout() {
      this.token = ''
      this.userId = 0
      this.userUuid = ''
      this.username = ''
      this.isAdmin = false
      localStorage.removeItem('token')
      localStorage.removeItem('userId')
      localStorage.removeItem('userUuid')
      localStorage.removeItem('username')
      localStorage.removeItem('isAdmin')
    },
    // 页面刷新后从 /auth/me 拉一次最新状态, 修复 localStorage 缺失字段导致的 isAdmin=false
    async refresh() {
      if (!this.token) return
      try {
        const res = await authApi.me()
        this.userId = res.data.id
        this.userUuid = res.data.uuid || ''
        this.username = res.data.username
        this.isAdmin = !!res.data.is_admin
        localStorage.setItem('userId', String(res.data.id))
        localStorage.setItem('userUuid', res.data.uuid || '')
        localStorage.setItem('username', res.data.username)
        localStorage.setItem('isAdmin', res.data.is_admin ? '1' : '0')
      } catch {
        // token 失效 / 网络问题 — 让 http 拦截器统一处理(401 会清掉 token)
      }
    },
  },
})
