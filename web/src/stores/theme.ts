import { defineStore } from 'pinia'

type Theme = 'light' | 'dark'
const KEY = 'zzdzz-demo-theme'

function detect(): Theme {
  const saved = localStorage.getItem(KEY) as Theme | null
  if (saved === 'light' || saved === 'dark') return saved
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return 'light'
}

function apply(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme)
}

export const useThemeStore = defineStore('theme', {
  state: () => ({ theme: detect() as Theme }),
  actions: {
    init() {
      apply(this.theme)
    },
    toggle() {
      this.theme = this.theme === 'light' ? 'dark' : 'light'
      localStorage.setItem(KEY, this.theme)
      apply(this.theme)
    },
    set(theme: Theme) {
      this.theme = theme
      localStorage.setItem(KEY, theme)
      apply(theme)
    },
  },
})
