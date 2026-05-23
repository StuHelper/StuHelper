<template>
  <main class="min-h-screen bg-bg-base px-4 py-8 flex items-center justify-center">
    <section class="w-full max-w-[420px] bg-bg-card border border-border rounded-lg shadow-soft p-8 text-center">
      <div v-if="loading" class="flex flex-col items-center gap-4">
        <div class="w-8 h-8 border-2 border-border border-t-primary rounded-full animate-spin" />
        <div>
          <h1 class="m-0 text-xl font-bold text-text-primary">{{ t('common.openPlatformAuthorize.title') }}</h1>
          <p class="mt-2 mb-0 text-sm text-text-muted">{{ t('common.openPlatformAuthorize.subtitle') }}</p>
        </div>
      </div>

      <div v-else class="flex flex-col items-center gap-4">
        <AlertTriangle class="w-10 h-10 text-danger" aria-hidden="true" />
        <div>
          <h1 class="m-0 text-xl font-bold text-text-primary">{{ t('common.openPlatformAuthorize.failed') }}</h1>
          <p class="mt-2 mb-0 text-sm text-text-muted">{{ error }}</p>
        </div>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AlertTriangle } from 'lucide-vue-next'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'

interface AuthorizeQuery {
  clientID: string
  redirectURI: string
  scope: string
  state?: string
}

const route = useRoute()
const { t } = useI18n()
const loading = ref(true)
const error = ref('')

function singleQuery(name: string): string {
  const value = route.query[name]
  return typeof value === 'string' ? value : ''
}

function authorizeQuery(): AuthorizeQuery {
  return {
    clientID: singleQuery('client_id'),
    redirectURI: singleQuery('redirect_uri'),
    scope: singleQuery('scope'),
    state: singleQuery('state') || undefined,
  }
}

function validateQuery(query: AuthorizeQuery) {
  if (!query.clientID || !query.redirectURI || !query.scope) {
    throw new Error(t('common.openPlatformAuthorize.invalidRequest'))
  }
}

function navigateTo(rawURL: string) {
  const parsed = new URL(rawURL, window.location.origin)
  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new Error('Invalid authorization redirect URL protocol')
  }
  window.location.assign(parsed.toString())
}

onMounted(async () => {
  try {
    const query = authorizeQuery()
    validateQuery(query)
    const response = await api.openPlatform.authorize(query)
    const data = response.data?.data
    const target = data?.redirectURL || data?.consentURL || data?.profileCompletionURL
    if (!target) throw new Error('Invalid authorization response')
    navigateTo(target)
  } catch (err) {
    error.value = getErrorMessage(err, t('common.openPlatformAuthorize.failed'))
    loading.value = false
  }
})
</script>
