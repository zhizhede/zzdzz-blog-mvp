<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{ source: string }>()

// breaks: 单个换行渲染为 <br>(GitHub 评论风格)。诗歌/歌词类内容每行
// 只有一个 \n 且无空行,标准 CommonMark 会把整篇合并成一个段落。
const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true,
})

const html = computed(() => md.render(props.source || ''))
</script>

<template>
  <div class="prose" v-html="html" />
</template>
