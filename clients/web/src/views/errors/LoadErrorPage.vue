<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-bg-base">
    <div class="text-center max-w-[400px]">
      <div class="w-20 h-20 mx-auto mb-6 text-text-muted opacity-40">
        <AlertCircle class="w-full h-full" />
      </div>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0 mb-3">{{ t('errors.loadError.title') }}</h1>
      <p class="text-sm text-text-muted m-0 mb-6">{{ t('errors.loadError.description') }}</p>
      <button
        class="inline-flex items-center gap-2 py-2 px-5 bg-gradient-to-br from-primary to-accent text-white text-sm font-medium rounded-full transition-all duration-fast hover:opacity-90 hover:-translate-y-px"
        @click="reload"
      >
        <RefreshCw :size="18" />
        {{ t('errors.loadError.reload') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle, RefreshCw } from 'lucide-vue-next'

const { t } = useI18n()

const reload = () => {
  window.location.reload()
}
</script>
