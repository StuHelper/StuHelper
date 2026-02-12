<template>
  <div class="min-h-[80vh] flex items-center justify-center animate-fade-in">
    <div class="text-center max-w-[360px]">
      <div class="w-2 h-2 rounded-full bg-primary mx-auto mb-4 animate-glow-pulse"></div>
      <h1 class="font-sans text-xl font-extrabold tracking-tight m-0 mb-3 gradient-text">{{ t('teaching.comingSoon') }}</h1>
      <p class="text-text-muted m-0 mb-8 text-sm">{{ moduleName }}</p>
      <button
        class="py-2 px-6 bg-gradient-to-br from-primary to-accent text-white border-none rounded-full text-sm font-medium cursor-pointer transition-all duration-fast hover:opacity-90 hover:-translate-y-px"
        @click="goBack"
      >
        {{ t('nav.home') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const moduleName = computed(() => {
  if (route.meta.titleKey) {
    return t(route.meta.titleKey as string)
  }
  return ''
})

const goBack = () => {
  router.push('/')
}
</script>
