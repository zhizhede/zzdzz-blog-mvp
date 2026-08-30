<script setup lang="ts">
import type { Tag } from '../api'

defineProps<{
  tag: Tag
  active?: boolean
  count?: number
}>()

defineEmits<{ (e: 'click', tag: Tag): void }>()
</script>

<template>
  <button
    :class="['tag-pill', { active }]"
    type="button"
    @click="$emit('click', tag)"
  >
    <span class="hash">#</span>
    <span class="name">{{ tag.name }}</span>
    <span v-if="count !== undefined" class="count">{{ count }}</span>
  </button>
</template>

<style scoped>
.tag-pill {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  border: 1px solid var(--rule-soft);
  background: transparent;
  color: var(--ink-soft);
  padding: 4px 10px;
  border-radius: 999px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  cursor: pointer;
  transition: all var(--transition);
}
.tag-pill .hash {
  color: var(--ink-mute);
}
.tag-pill .count {
  color: var(--ink-faint);
  font-size: 10px;
}
.tag-pill:hover {
  color: var(--ink);
  border-color: var(--ink-mute);
}
.tag-pill.active {
  background: var(--ink);
  color: var(--ink-on-inverse);
  border-color: var(--ink);
}
.tag-pill.active .hash,
.tag-pill.active .count {
  color: var(--ink-on-inverse);
  opacity: 0.7;
}
</style>
