import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userId: Number(localStorage.getItem('userId') || 0),
    username: localStorage.getItem('username') || '',
  }),
  actions: {
    setAuth(token: string, userId: number, username: string) {
      this.token = token
      this.userId = userId
      this.username = username
      localStorage.setItem('token', token)
      localStorage.setItem('userId', String(userId))
      localStorage.setItem('username', username)
    },
    logout() {
      this.token = ''
      this.userId = 0
      this.username = ''
      localStorage.removeItem('token')
      localStorage.removeItem('userId')
      localStorage.removeItem('username')
    },
  },
})