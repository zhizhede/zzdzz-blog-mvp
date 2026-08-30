<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { tagApi, type TagWithCount } from '../api'
import TagPill from './TagPill.vue'

const props = defineProps<{ activeTagId?: number | null }>()
const emit = defineEmits<{ (e: 'select', tagId: number | null): void }>()

const tags = ref<TagWithCount[]>([])
onMounted(async () => {
  const res = await tagApi.list()
  tags.value = res.data ?? []
})

const ranked = computed(() => tags.value)
const max = computed(() => Math.max(1, ...ranked.value.map((r) => r.count)))
</script>

<template>
  <div class="tag-cloud">
    <TagPill
      :tag="{ id: 0, name: '全部', slug: 'all' }"
      :active="!props.activeTagId"
      @click="emit('select', null)"
    />
    <TagPill
      v-for="t in ranked"
      :key="t.id"
      :tag="t"
      :count="t.count"
      :active="props.activeTagId === t.id"
      :style="{ fontSize: 11 + (t.count / max) * 6 + 'px' }"
      @click="emit('select', t.id)"
    />
  </div>
</template>

<style scoped>
.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
}
</style>