<template>
  <div class="infinite-scroll">
    <slot />

    <div ref="sentinelRef" class="sentinel">
      <div v-if="loading" class="loading-indicator">
        <span class="spinner" />
        <span>{{ loadingText ?? t('common.actions.loading') }}</span>
      </div>
      <div v-else-if="error" class="error">
        <span>{{ errorText ?? t('common.login.loadFailed') }}</span>
        <button @click="$emit('retry')" class="retry-btn">{{ t('common.actions.retry') }}</button>
      </div>
      <div v-else-if="!hasMore" class="no-more">
        {{ noMoreText ?? t('common.login.noMore') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  loading?: boolean
  hasMore?: boolean
  error?: boolean
  loadingText?: string
  noMoreText?: string
  errorText?: string
  threshold?: number
}>(), {
  loading: false,
  hasMore: true,
  error: false,
  loadingText: undefined,
  noMoreText: undefined,
  errorText: undefined,
  threshold: 100
})

const emit = defineEmits<{
  loadMore: []
  retry: []
}>()

const sentinelRef = ref<HTMLElement>()
let observer: IntersectionObserver | null = null

onMounted(() => {
  if (!sentinelRef.value) return

  observer = new IntersectionObserver(
    (entries) => {
      const entry = entries[0]
      if (entry.isIntersecting && !props.loading && props.hasMore && !props.error) {
        emit('loadMore')
      }
    },
    {
      rootMargin: `${props.threshold}px`
    }
  )

  observer.observe(sentinelRef.value)
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<style scoped>
.infinite-scroll {
  width: 100%;
}

.sentinel {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: var(--space-4);
  min-height: 60px;
}

.loading-indicator {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--text-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.no-more {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.error {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.retry-btn {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.retry-btn:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
}
</style>
