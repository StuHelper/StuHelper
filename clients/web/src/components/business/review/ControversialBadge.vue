<template>
  <div
    v-if="dislikeCount >= threshold"
    class="flex items-center gap-2 py-1.5 px-3 rounded-md bg-warning/10"
  >
    <TriangleAlert :size="14" class="shrink-0 text-warning" />
    <span class="text-xs font-medium text-warning">
      {{ t('review.card.controversial') }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { TriangleAlert } from 'lucide-vue-next'

const { t } = useI18n()

withDefaults(defineProps<{
  dislikeCount: number
  threshold?: number
}>(), {
  threshold: 5,
})
</script>
