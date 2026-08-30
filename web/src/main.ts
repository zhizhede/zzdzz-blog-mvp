import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import './styles/tokens.css'
import './styles/reset.css'
import './styles/global.css'

import { useThemeStore } from './stores/theme'
import { useUserStore } from './stores/user'

const app = createApp(App)

for (const [key, comp] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, comp as any)
}

app.use(createPinia())
useThemeStore().init()
app.use(router)
app.use(ElementPlus)

// 页面刷新后从 /auth/me 同步一次最新状态, 修复旧版 localStorage 缺字段导致的权限判断错误
useUserStore().refresh()

app.mount('#app')