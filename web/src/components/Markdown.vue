<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{ source: string }>()

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: false,
})

const html = computed(() => md.render(props.source || ''))
</script>

<template>
  <div class="prose" v-html="html" />
</template>
