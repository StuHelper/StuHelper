<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Props {
  type?: 'card' | 'list' | 'text'
  count?: number
}

withDefaults(defineProps<Props>(), {
  type: 'card',
  count: 3
})
</script>

<template>
  <div role="status" aria-busy="true" :aria-label="t('common.loading.label')" class="space-y-4">
    <template v-if="type === 'card'">
      <div v-for="i in count" :key="i" class="skeleton-shimmer bg-gray-200 dark:bg-gray-700 rounded-2xl h-48 w-full"></div>
    </template>
    <template v-else-if="type === 'list'">
      <div v-for="i in count" :key="i" class="flex space-x-4">
        <div class="skeleton-shimmer rounded-full bg-gray-200 dark:bg-gray-700 h-12 w-12 flex-shrink-0"></div>
        <div class="flex-1 space-y-3 py-1">
          <div class="skeleton-shimmer h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
          <div class="skeleton-shimmer h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
        </div>
      </div>
    </template>
    <template v-else>
      <div v-for="i in count" :key="i" class="space-y-2">
        <div class="skeleton-shimmer h-4 bg-gray-200 dark:bg-gray-700 rounded w-full"></div>
        <div class="skeleton-shimmer h-4 bg-gray-200 dark:bg-gray-700 rounded w-5/6"></div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.skeleton-shimmer {
  position: relative;
  overflow: hidden;
}

.skeleton-shimmer::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.5) 50%,
    transparent 100%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
</style>
