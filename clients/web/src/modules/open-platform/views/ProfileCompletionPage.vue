<template>
  <main class="min-h-screen bg-bg-base px-4 py-8 flex items-center justify-center">
    <section class="w-full max-w-[720px] bg-bg-card border border-border rounded-lg shadow-soft overflow-hidden">
      <div class="h-1 bg-gradient-to-r from-primary via-secondary to-accent" />

      <div v-if="loading" class="min-h-[360px] flex flex-col items-center justify-center gap-4 text-text-muted">
        <div class="w-8 h-8 border-2 border-border border-t-primary rounded-full animate-spin" />
        <p class="text-sm">{{ t('common.openPlatformProfileCompletion.loading') }}</p>
      </div>

      <div v-else-if="error" class="min-h-[360px] flex flex-col items-center justify-center gap-4 px-6 text-center">
        <AlertTriangle class="w-10 h-10 text-danger" aria-hidden="true" />
        <div>
          <h1 class="m-0 text-xl font-bold text-text-primary">
            {{ errorTitle }}
          </h1>
          <p class="mt-2 mb-0 text-sm text-text-muted">{{ error }}</p>
        </div>
        <button
          v-ripple
          class="inline-flex items-center gap-2 px-4 py-2 rounded-md border border-border bg-bg-elevated text-text-primary text-sm font-medium cursor-pointer hover:bg-bg-hover disabled:opacity-50 disabled:cursor-not-allowed"
          type="button"
          :disabled="!token || loading"
          @click="loadCompletion"
        >
          <RefreshCw class="w-4 h-4" aria-hidden="true" />
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <div v-else-if="completion" class="p-6 sm:p-8">
        <header class="flex flex-col gap-5 sm:flex-row sm:items-start">
          <div class="w-12 h-12 rounded-lg bg-primary-alpha text-primary flex items-center justify-center shrink-0">
            <UserCheck class="w-6 h-6" aria-hidden="true" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="m-0 text-xs font-semibold uppercase text-text-muted">StuHelper Identity</p>
            <h1 class="mt-2 mb-0 text-2xl font-extrabold text-text-primary leading-tight">
              {{ t('common.openPlatformProfileCompletion.title', { app: completion.app.displayName }) }}
            </h1>
            <p v-if="actingUser" class="mt-2 mb-0 text-sm text-text-secondary">
              {{ t('common.openPlatformProfileCompletion.identityLine', { user: actingUser }) }}
            </p>
          </div>
        </header>

        <section class="mt-7 border-t border-border pt-6">
          <div class="flex items-center justify-between gap-4">
            <h2 class="m-0 text-sm font-semibold text-text-primary">
              {{ t('common.openPlatformProfileCompletion.requiredFields') }}
            </h2>
            <span class="text-xs text-text-muted">
              {{ t('common.openPlatformProfileCompletion.expiresAt', { time: expiresAtText }) }}
            </span>
          </div>

          <div v-if="completion.missingFields.length > 0" class="mt-4 border border-border rounded-lg overflow-hidden">
            <article
              v-for="field in completion.missingFields"
              :key="field.key"
              class="completion-row"
            >
              <div class="min-w-0">
                <h3 class="m-0 text-sm font-semibold text-text-primary">{{ field.displayName }}</h3>
                <p class="mt-1 mb-0 text-xs text-text-muted">{{ field.key }}</p>
              </div>
              <a
                class="completion-action"
                :href="field.actionURL"
                target="_blank"
                rel="noreferrer"
              >
                {{ t('common.openPlatformProfileCompletion.openAction') }}
                <ExternalLink class="w-3.5 h-3.5" aria-hidden="true" />
              </a>
            </article>
          </div>

          <p v-else class="mt-4 mb-0 text-sm text-text-secondary">
            {{ t('common.openPlatformProfileCompletion.noMissingFields') }}
          </p>
        </section>

        <section class="mt-7 border-t border-border pt-6">
          <h2 class="m-0 text-sm font-semibold text-text-primary">
            {{ t('common.openPlatformProfileCompletion.requestedPermissions') }}
          </h2>
          <ul class="mt-4 mb-0 pl-0 list-none grid gap-2">
            <li
              v-for="scope in completion.scopes"
              :key="scope.scope"
              class="grid gap-1 rounded-md bg-bg-elevated p-3 text-sm sm:grid-cols-[1fr_auto] sm:items-start"
            >
              <span class="min-w-0">
                <span class="block font-medium text-text-primary">{{ scope.displayName }}</span>
                <span v-if="scope.reason" class="mt-1 block text-xs leading-relaxed text-text-secondary">
                  {{ t('common.openPlatformProfileCompletion.reason', { reason: scope.reason }) }}
                </span>
              </span>
              <span class="break-all text-xs text-text-muted sm:text-right">{{ scope.scope }}</span>
            </li>
          </ul>
        </section>

        <footer class="mt-8 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <button
            v-ripple
            class="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-md border border-border bg-bg-elevated text-text-primary text-sm font-semibold cursor-pointer hover:bg-bg-hover disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
            :disabled="submitting || loading"
            @click="loadCompletion"
          >
            <RefreshCw class="w-4 h-4" aria-hidden="true" />
            {{ t('common.openPlatformProfileCompletion.refresh') }}
          </button>
          <button
            v-ripple
            class="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-md border border-transparent bg-primary text-white text-sm font-semibold cursor-pointer hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
            :disabled="submitting"
            @click="continueAuthorization"
          >
            <ArrowRight class="w-4 h-4" aria-hidden="true" />
            {{ submitting ? t('common.openPlatformProfileCompletion.continuing') : t('common.openPlatformProfileCompletion.continue') }}
          </button>
        </footer>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, ArrowRight, ExternalLink, RefreshCw, UserCheck } from 'lucide-vue-next'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useAuthStore } from '@/stores/auth'
import type { OpenPlatformProfileCompletionPageResponse } from '@stuhelper/shared/api'

const route = useRoute()
const authStore = useAuthStore()
const { t } = useI18n()

const completion = ref<OpenPlatformProfileCompletionPageResponse | null>(null)
const errorTitle = ref('')
const error = ref('')
const loading = ref(true)
const submitting = ref(false)

const token = computed(() => {
  const value = route.query.token
  return typeof value === 'string' ? value : ''
})

const actingUser = computed(() => {
  return authStore.user?.displayName || authStore.user?.name || ''
})

const expiresAtText = computed(() => {
  if (!completion.value) return ''
  return new Date(completion.value.expiresAt).toLocaleString()
})

async function loadCompletion() {
  if (!token.value) {
    errorTitle.value = t('common.openPlatformProfileCompletion.loadFailed')
    error.value = t('common.openPlatformProfileCompletion.invalidToken')
    loading.value = false
    return
  }
  errorTitle.value = ''
  error.value = ''
  loading.value = true
  try {
    const response = await api.openPlatform.getProfileCompletion(token.value)
    const data = response.data?.data
    if (!data) throw new Error('Invalid profile completion response')
    completion.value = data
  } catch (err) {
    errorTitle.value = t('common.openPlatformProfileCompletion.loadFailed')
    error.value = getErrorMessage(err, t('common.openPlatformProfileCompletion.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function continueAuthorization() {
  if (!completion.value || submitting.value) return
  submitting.value = true
  try {
    const response = await api.openPlatform.continueProfileCompletion(completion.value.token)
    const data = response.data?.data
    const target = data?.redirectURL || data?.consentURL || data?.profileCompletionURL
    if (!target) throw new Error('Invalid authorization response')
    redirectToURL(target)
  } catch (err) {
    errorTitle.value = t('common.openPlatformProfileCompletion.submitFailedTitle')
    error.value = getErrorMessage(err, t('common.openPlatformProfileCompletion.submitFailed'))
    submitting.value = false
  }
}

function redirectToURL(rawURL: string) {
  const parsed = new URL(rawURL, window.location.origin)
  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new Error('Invalid redirect URL protocol')
  }
  window.location.assign(parsed.toString())
}

onMounted(loadCompletion)
</script>

<style scoped>
.completion-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem;
  border-top: 1px solid var(--color-border);
}

.completion-row:first-child {
  border-top: 0;
}

.completion-action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  flex-shrink: 0;
  color: var(--color-primary);
  font-size: 0.875rem;
  font-weight: 700;
  text-decoration: none;
}

.completion-action:hover {
  color: var(--color-primary-dark);
}
</style>
