import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// SSE 长连接必须 HTTP/1.1,Connection 用 keep-alive,
// 否则部分代理会把响应缓冲到上游关闭后才一次性发给浏览器,看起来就是"全量输出"
const sseProxy = {
  target: 'http://localhost:8080',
  changeOrigin: true,
  configure(proxy: any) {
    proxy.on('proxyReq', (proxyReq: any) => {
      proxyReq.setHeader('Connection', 'keep-alive')
    })
  },
}

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      // AI 流式端点单独走 SSE-friendly 配置
      '/api/v1/ai': sseProxy,
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})