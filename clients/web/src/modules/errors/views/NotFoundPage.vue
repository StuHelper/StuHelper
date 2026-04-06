<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-bg-base">
    <div class="text-center max-w-[400px]">
      <div class="w-[120px] h-[120px] mx-auto mb-6 text-text-muted opacity-40">
        <svg viewBox="0 0 120 120" fill="none">
          <circle cx="60" cy="60" r="56" stroke="currentColor" stroke-width="2" opacity="0.2"/>
          <circle cx="60" cy="60" r="40" stroke="currentColor" stroke-width="2" opacity="0.3"/>
          <text x="60" y="72" text-anchor="middle" fill="currentColor" font-size="32" font-weight="700">404</text>
        </svg>
      </div>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0 mb-3">{{ t('errors.notFound.title') }}</h1>
      <p class="text-sm text-text-muted m-0 mb-8">{{ t('errors.notFound.description') }}</p>
      <div class="flex gap-3 justify-center max-sm:flex-col">
        <router-link
          to="/"
          class="inline-flex items-center gap-2 py-2 px-5 text-sm font-medium rounded-full no-underline transition-all duration-fast bg-gradient-to-br from-primary to-accent text-white hover:opacity-90 hover:-translate-y-px max-sm:w-full max-sm:justify-center"
        >
          <Home :size="18" />
          {{ t('errors.notFound.backHome') }}
        </router-link>
        <button
          class="inline-flex items-center gap-2 py-2 px-5 text-sm font-medium rounded-full bg-transparent text-text-secondary transition-all duration-fast hover:border-text-primary hover:text-text-primary max-sm:w-full max-sm:justify-center"
          @click="goBack"
        >
          <ArrowLeft :size="18" />
          {{ t('errors.notFound.goBack') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Home, ArrowLeft } from 'lucide-vue-next'

const router = useRouter()
const { t } = useI18n()

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/')
  }
}
</script>
