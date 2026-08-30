<script setup lang="ts">
const year = new Date().getFullYear()
</script>

<template>
  <footer class="app-footer">
    <div class="container footer-inner">
      <span class="mono">© {{ year }} zzdzz blog · demo build</span>
      <span class="dot" />
      <span class="mono">独立 Vite + Vue3 项目,数据全在内存</span>
    </div>
  </footer>
</template>

<style scoped>
.app-footer {
  border-top: 1px solid var(--rule);
  margin-top: 80px;
  background: var(--bg);
}
.footer-inner {
  padding: 24px 24px 32px;
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}
.dot {
  width: 3px;
  height: 3px;
  background: var(--ink-mute);
  border-radius: 50%;
}
</style>
