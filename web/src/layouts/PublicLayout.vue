<script setup lang="ts">
import AppHeader from '../components/AppHeader.vue'
import AppFooter from '../components/AppFooter.vue'
</script>

<template>
  <div class="public-layout">
    <AppHeader />
    <main class="public-main">
      <router-view />
    </main>
    <AppFooter />
  </div>
</template>

<style scoped>
.public-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.public-main {
  flex: 1;
}
</style>