<template>
  <div class="min-h-screen flex items-center justify-center bg-bg-base">
    <div class="bg-bg-card py-12 px-10 rounded-md text-center max-w-[360px] w-full">
      <!-- 加载中 -->
      <div v-if="loading" class="flex flex-col items-center gap-4 text-text-muted text-sm">
        <div class="w-8 h-8 border-2 border-border border-t-accent rounded-full animate-spin"></div>
        <p>{{ t('errors.authCallback.loading') }}</p>
      </div>

      <!-- 通用错误 -->
      <div v-else-if="error" class="text-text-primary">
        <p>{{ t('errors.authCallback.error') }}</p>
        <p class="text-text-muted text-sm my-3 mb-6">{{ error }}</p>
        <button
          class="py-2 px-6 bg-text-primary text-bg-base border-none rounded-sm text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white"
          @click="goToLogin"
        >
          {{ t('errors.authCallback.backToLogin') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import type { LocationQueryValue } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { resolveApiURL } from '@/api/client'
import { consumeOAuthState } from '@/utils/auth'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const loading = ref(true)
const error = ref('')

function getSingleQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined): string {
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '')
}

function buildBackendCallbackURL(code: string, state: string): string {
  const url = resolveApiURL('/api/v1/auth/callback')
  url.searchParams.set('code', code)
  url.searchParams.set('state', state)
  return url.toString()
}

onMounted(async () => {
  const code = getSingleQueryValue(route.query.code)
  const state = getSingleQueryValue(route.query.state)

  if (!code) {
    error.value = t('errors.authCallback.missingCode')
    loading.value = false
    return
  }

  if (!state) {
    error.value = t('errors.authCallback.missingState')
    loading.value = false
    return
  }

  if (!consumeOAuthState(state)) {
    error.value = t('common.login.invalidState')
    loading.value = false
    return
  }

  window.location.replace(buildBackendCallbackURL(code, state))
})

const goToLogin = () => {
  router.push('/login')
}
</script>
