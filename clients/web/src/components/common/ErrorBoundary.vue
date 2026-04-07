<template>
  <div v-if="error" class="min-h-[60vh] flex flex-col items-center justify-center gap-4 p-8 text-center">
    <div class="w-14 h-14 rounded-full bg-danger/10 flex items-center justify-center">
      <AlertTriangle :size="28" class="text-danger" />
    </div>
    <h2 class="text-lg font-semibold text-text-primary">{{ t('errors.boundary.title') }}</h2>
    <p class="text-sm text-text-secondary max-w-md">{{ t('errors.boundary.description') }}</p>
    <button
      class="mt-2 py-2 px-6 text-sm font-medium text-white bg-primary rounded-full cursor-pointer transition-opacity duration-fast hover:opacity-90"
      @click="handleReload"
    >
      {{ t('errors.boundary.reload') }}
    </button>
  </div>
  <slot v-else />
</template>

<script setup lang="ts">
import { ref, onErrorCaptured, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AlertTriangle } from 'lucide-vue-next'

const { t } = useI18n()
const route = useRoute()
const error = ref<Error | null>(null)

// 路由切换时自动清除错误状态，恢复正常渲染
watch(() => route.fullPath, () => {
  if (error.value) {
    error.value = null
    retryCount = 0
  }
})

onErrorCaptured((err: Error) => {
  if (import.meta.env.DEV) {
    console.error('[ErrorBoundary] Captured error:', err)
  }
  error.value = err
  // 返回 false 阻止错误继续向上传播
  return false
})

// 先尝试组件级恢复（清除错误状态重新渲染），失败后再整页刷新
let retryCount = 0
const MAX_RETRY = 2

function handleReload() {
  if (retryCount < MAX_RETRY) {
    retryCount++
    error.value = null
  } else {
    retryCount = 0
    error.value = null
    window.location.reload()
  }
}
</script>
