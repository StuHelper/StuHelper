<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-bg-base font-sans">
    <div class="text-center max-w-[400px]">
      <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="mx-auto mb-6 text-text-muted">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <h1 class="text-xl font-bold text-text-primary mb-3">{{ t('errors.loadError.title') }}</h1>
      <p class="text-sm text-text-secondary mb-6">{{ t('errors.loadError.description') }}</p>
      <button
        class="inline-flex items-center gap-2 px-6 py-2.5 bg-gradient-to-br from-indigo-500 to-violet-500 text-white text-sm font-medium border-none rounded-full cursor-pointer hover:opacity-90 transition-opacity"
        @click="reload"
      >
        {{ t('errors.loadError.reload') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const reload = () => {
  // chunk 错误页的重试依赖浏览器重载当前页面，不属于业务 API 请求。
  window.location.reload()
}
</script>
