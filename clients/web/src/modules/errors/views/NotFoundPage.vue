<template>
  <div
    class="min-h-screen flex items-center justify-center p-6 bg-bg-base"
    :data-join-admission-not-found="isJoinAdmissionNotFound ? 'true' : undefined"
  >
    <div class="text-center max-w-[400px]">
      <div class="w-[120px] h-[120px] mx-auto mb-6 text-text-muted opacity-40">
        <svg viewBox="0 0 120 120" fill="none">
          <circle cx="60" cy="60" r="56" stroke="currentColor" stroke-width="2" opacity="0.2"/>
          <circle cx="60" cy="60" r="40" stroke="currentColor" stroke-width="2" opacity="0.3"/>
          <text x="60" y="72" text-anchor="middle" fill="currentColor" font-size="32" font-weight="700">404</text>
        </svg>
      </div>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0 mb-3">{{ pageCopy.title }}</h1>
      <p class="text-sm text-text-muted m-0 mb-8">{{ pageCopy.description }}</p>
      <div class="flex gap-3 justify-center max-sm:flex-col">
        <a
          :href="homeHref"
          class="inline-flex items-center gap-2 py-2 px-5 text-sm font-medium rounded-full no-underline transition-all duration-fast bg-gradient-to-br from-primary to-accent text-white hover:opacity-90 hover:-translate-y-px max-sm:w-full max-sm:justify-center"
        >
          <Home :size="18" />
          {{ pageCopy.backHome }}
        </a>
        <button
          class="inline-flex items-center gap-2 py-2 px-5 text-sm font-medium rounded-full bg-transparent text-text-secondary transition-all duration-fast hover:border-text-primary hover:text-text-primary max-sm:w-full max-sm:justify-center"
          @click="goBack"
        >
          <ArrowLeft :size="18" />
          {{ pageCopy.goBack }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Home, ArrowLeft } from 'lucide-vue-next'
import { updatePageMeta } from '@/composables/usePageMeta'
import { currentHostname, isJoinAdmissionHost } from '@/router/join-domain'
import { accountCenterURL } from '@/utils/redirect'

const router = useRouter()
const { t } = useI18n()
const homeHref = computed(() => accountCenterURL('/'))
const isJoinAdmissionNotFound = computed(() => isJoinAdmissionHost(currentHostname()))
const pageCopy = computed(() => {
  const key = isJoinAdmissionNotFound.value
    ? 'errors.notFound.joinAdmission'
    : 'errors.notFound'
  return {
    title: t(`${key}.title`),
    description: t(`${key}.description`),
    backHome: t(`${key}.backHome`),
    goBack: t(`${key}.goBack`),
  }
})

watchEffect(() => {
  updatePageMeta({
    title: pageCopy.value.title,
    description: pageCopy.value.description,
  })
})

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    window.location.assign(homeHref.value)
  }
}
</script>
