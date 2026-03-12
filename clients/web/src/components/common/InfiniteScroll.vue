<template>
  <div class="w-full">
    <slot />

    <div ref="sentinelRef" class="flex justify-center items-center p-4 min-h-[60px]" aria-live="polite" role="status">
      <div v-if="loading" class="flex items-center gap-2 text-text-muted text-sm">
        <span class="inline-block w-5 h-5 border-2 border-border border-t-text-primary rounded-full animate-spin" />
        <span>{{ loadingText ?? t('common.actions.loading') }}</span>
      </div>
      <div v-else-if="error" class="flex items-center gap-3 text-text-muted text-sm">
        <span>{{ errorText ?? t('common.loadFailed') }}</span>
        <button
          class="px-3 py-1 bg-transparent border border-border rounded-sm text-text-secondary text-sm cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
          @click="$emit('retry')"
        >
          {{ t('common.actions.retry') }}
        </button>
      </div>
      <div v-else-if="!hasMore" class="text-text-muted text-sm">
        {{ noMoreText ?? t('common.noMore') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
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

const setupObserver = () => {
  // M-78: disconnect 后置空，释放引用
  if (observer) { observer.disconnect(); observer = null }
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
}

onMounted(setupObserver)

// threshold 变化时重建 observer（IntersectionObserver 选项创建后不可变）
watch(() => props.threshold, setupObserver)

// L-04: loading 结束后若 sentinel 仍在视口内，手动触发下一次加载
watch(() => props.loading, (newVal, oldVal) => {
  if (oldVal && !newVal && props.hasMore && !props.error && sentinelRef.value) {
    const rect = sentinelRef.value.getBoundingClientRect()
    if (rect.top < window.innerHeight + props.threshold) {
      emit('loadMore')
    }
  }
})

onUnmounted(() => {
  if (observer) { observer.disconnect(); observer = null }
})
</script>
