<script setup lang="ts">
defineProps<{
  prefix?: string
  text: string
  suffix?: string
}>()
</script>

<template>
  <span class="issue-tag">
    <span v-if="prefix">{{ prefix }}</span>
    <span>{{ text }}</span>
    <span v-if="suffix" class="dot" />
    <span v-if="suffix">{{ suffix }}</span>
  </span>
</template>
