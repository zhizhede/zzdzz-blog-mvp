<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="blog">
    <header class="blog-header">
      <div class="blog-header-inner">
        <h1 class="brand" @click="router.push('/blog')">zzdzz blog</h1>
        <nav>
          <router-link to="/admin/articles" v-if="$route.path.startsWith('/blog')">后台</router-link>
        </nav>
      </div>
    </header>
    <main class="blog-main">
      <router-view />
    </main>
    <footer class="blog-footer">© 2026 zzdzz blog · powered by Go + Vue3</footer>
  </div>
</template>

<style scoped>
.blog { min-height: 100vh; display: flex; flex-direction: column; }
.blog-header {
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.blog-header-inner {
  max-width: 800px; margin: 0 auto; padding: 16px 24px;
  display: flex; align-items: center; justify-content: space-between;
}
.brand { margin: 0; font-size: 20px; cursor: pointer; letter-spacing: 1px; }
.blog-main { flex: 1; max-width: 800px; margin: 0 auto; padding: 32px 24px; width: 100%; box-sizing: border-box; }
.blog-footer { text-align: center; padding: 20px; color: #909399; border-top: 1px solid #ebeef5; background: #fff; }
a { color: #409eff; text-decoration: none; }
</style>