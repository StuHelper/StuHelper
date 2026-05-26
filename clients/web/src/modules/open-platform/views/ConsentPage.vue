<template>
  <main class="min-h-screen bg-bg-base px-4 py-8 flex items-center justify-center">
    <section class="w-full max-w-[720px] bg-bg-card border border-border rounded-lg shadow-soft overflow-hidden">
      <div class="h-1 bg-gradient-to-r from-primary via-secondary to-accent" />

      <div v-if="loading" class="min-h-[360px] flex flex-col items-center justify-center gap-4 text-text-muted">
        <div class="w-8 h-8 border-2 border-border border-t-primary rounded-full animate-spin" />
        <p class="text-sm">{{ t('common.openPlatformConsent.loading') }}</p>
      </div>

      <div v-else-if="error" class="min-h-[360px] flex flex-col items-center justify-center gap-4 px-6 text-center">
        <AlertTriangle class="w-10 h-10 text-danger" aria-hidden="true" />
        <div>
          <h1 class="m-0 text-xl font-bold text-text-primary">{{ errorTitle }}</h1>
          <p class="mt-2 mb-0 text-sm text-text-muted">{{ error }}</p>
        </div>
        <button
          v-ripple
          class="inline-flex items-center gap-2 px-4 py-2 rounded-md border border-border bg-bg-elevated text-text-primary text-sm font-medium cursor-pointer hover:bg-bg-hover disabled:opacity-50 disabled:cursor-not-allowed"
          type="button"
          :disabled="!token || loading"
          @click="loadConsent"
        >
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <div v-else-if="consent" class="p-6 sm:p-8">
        <header class="flex flex-col gap-5 sm:flex-row sm:items-start">
          <div class="w-12 h-12 rounded-lg bg-primary-alpha text-primary flex items-center justify-center shrink-0">
            <ShieldCheck class="w-6 h-6" aria-hidden="true" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="m-0 text-xs font-semibold uppercase text-text-muted">StuHelper Identity</p>
            <h1 class="mt-2 mb-0 text-2xl font-extrabold text-text-primary leading-tight">
              {{ t('common.openPlatformConsent.title', { app: consent.app.displayName }) }}
            </h1>
            <p v-if="actingUser" class="mt-2 mb-0 text-sm text-text-secondary">
              {{ t('common.openPlatformConsent.identityLine', { user: actingUser }) }}
            </p>
          </div>
        </header>

        <section class="mt-7 border-t border-border pt-6">
          <h2 class="m-0 text-sm font-semibold text-text-primary">{{ t('common.openPlatformConsent.appInfo') }}</h2>
          <dl class="mt-4 grid gap-3 text-sm">
            <div class="grid grid-cols-[88px_1fr] gap-3">
              <dt class="text-text-muted">{{ t('common.openPlatformConsent.appName') }}</dt>
              <dd class="m-0 text-text-primary font-medium break-words">{{ consent.app.displayName }}</dd>
            </div>
            <div v-if="consent.app.description" class="grid grid-cols-[88px_1fr] gap-3">
              <dt class="text-text-muted">{{ t('common.openPlatformConsent.description') }}</dt>
              <dd class="m-0 text-text-secondary leading-relaxed">{{ consent.app.description }}</dd>
            </div>
            <div class="grid grid-cols-[88px_1fr] gap-3">
              <dt class="text-text-muted">{{ t('common.openPlatformConsent.redirect') }}</dt>
              <dd class="m-0 text-text-primary break-all">{{ redirectHost }}</dd>
            </div>
            <div class="grid grid-cols-[88px_1fr] gap-3">
              <dt class="text-text-muted">{{ t('common.openPlatformConsent.links') }}</dt>
              <dd class="m-0 flex flex-wrap gap-3">
                <a class="consent-link" :href="consent.app.homepageURL" target="_blank" rel="noreferrer">
                  {{ t('common.openPlatformConsent.homepage') }}
                  <ExternalLink class="w-3.5 h-3.5" aria-hidden="true" />
                </a>
                <a class="consent-link" :href="consent.app.privacyPolicyURL" target="_blank" rel="noreferrer">
                  {{ t('common.openPlatformConsent.privacy') }}
                  <ExternalLink class="w-3.5 h-3.5" aria-hidden="true" />
                </a>
              </dd>
            </div>
          </dl>
        </section>

        <section class="mt-7 border-t border-border pt-6">
          <div class="flex items-center justify-between gap-4">
            <h2 class="m-0 text-sm font-semibold text-text-primary">
              {{ t('common.openPlatformConsent.permissions') }}
            </h2>
            <span class="text-xs text-text-muted">
              {{ t('common.openPlatformConsent.expiresAt', { time: expiresAtText }) }}
            </span>
          </div>

          <div class="mt-4 border border-border rounded-lg overflow-hidden">
            <article
              v-for="scope in consent.scopes"
              :key="scope.scope"
              class="permission-row"
            >
              <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <h3 class="m-0 text-sm font-semibold text-text-primary">{{ scope.displayName }}</h3>
                  <p class="mt-1 mb-0 text-xs text-text-muted">{{ scope.scope }}</p>
                </div>
                <span class="sensitivity-badge" :class="sensitivityClass(scope.sensitivity)">
                  {{ sensitivityLabel(scope.sensitivity) }}
                </span>
              </div>
              <ul class="mt-3 mb-0 pl-0 list-none grid gap-2">
                <li
                  v-for="field in scope.fields"
                  :key="`${scope.scope}:${field}`"
                  class="flex items-center gap-2 text-sm text-text-secondary"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-secondary shrink-0" aria-hidden="true" />
                  <span>{{ field }}</span>
                </li>
              </ul>
              <p v-if="scope.reason" class="mt-3 mb-0 text-sm leading-relaxed text-text-secondary">
                {{ t('common.openPlatformConsent.reason', { reason: scope.reason }) }}
              </p>
            </article>
          </div>
        </section>

        <footer class="mt-8 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <button
            v-ripple
            class="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-md border border-border bg-bg-elevated text-text-primary text-sm font-semibold cursor-pointer hover:bg-bg-hover disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
            :disabled="submitting !== ''"
            @click="submitDecision(false)"
          >
            <X class="w-4 h-4" aria-hidden="true" />
            {{ submitting === 'deny' ? t('common.openPlatformConsent.denying') : t('common.openPlatformConsent.deny') }}
          </button>
          <button
            v-ripple
            class="inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-md border border-transparent bg-primary text-white text-sm font-semibold cursor-pointer hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
            :disabled="submitting !== ''"
            @click="submitDecision(true)"
          >
            <Check class="w-4 h-4" aria-hidden="true" />
            {{ submitting === 'accept' ? t('common.openPlatformConsent.accepting') : t('common.openPlatformConsent.accept') }}
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
import { AlertTriangle, Check, ExternalLink, ShieldCheck, X } from 'lucide-vue-next'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useAuthStore } from '@/stores/auth'
import {
  readConsentPagePayload,
  readRedirectURLPayload,
} from '@/modules/open-platform/pagePayload'
import type { OpenPlatformConsentPageResponse } from '@stuhelper/shared/api'

type Sensitivity = OpenPlatformConsentPageResponse['scopes'][number]['sensitivity']
type SubmitAction = 'accept' | 'deny'

const route = useRoute()
const authStore = useAuthStore()
const { t } = useI18n()

const consent = ref<OpenPlatformConsentPageResponse | null>(null)
const errorTitle = ref('')
const error = ref('')
const loading = ref(true)
const submitting = ref<SubmitAction | ''>('')

const token = computed(() => {
  const value = route.query.token
  return typeof value === 'string' ? value : ''
})

const actingUser = computed(() => {
  return authStore.user?.displayName || authStore.user?.name || ''
})

const redirectHost = computed(() => {
  if (!consent.value) return ''
  return new URL(consent.value.redirectURI).host
})

const expiresAtText = computed(() => {
  if (!consent.value) return ''
  return new Date(consent.value.expiresAt).toLocaleString()
})

async function loadConsent() {
  if (!token.value) {
    errorTitle.value = t('common.openPlatformConsent.loadFailed')
    error.value = t('common.openPlatformConsent.invalidToken')
    loading.value = false
    return
  }
  errorTitle.value = ''
  error.value = ''
  loading.value = true
  try {
    const response = await api.openPlatform.getConsent(token.value)
    consent.value = readConsentPagePayload(response.data?.data)
  } catch (err) {
    errorTitle.value = t('common.openPlatformConsent.loadFailed')
    error.value = getErrorMessage(err, t('common.openPlatformConsent.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function submitDecision(accept: boolean) {
  if (!consent.value || submitting.value) return
  submitting.value = accept ? 'accept' : 'deny'
  try {
    const action = accept ? api.openPlatform.acceptConsent : api.openPlatform.denyConsent
    const response = await action(consent.value.token)
    redirectToURL(readRedirectURLPayload(response.data?.data))
  } catch (err) {
    errorTitle.value = t('common.openPlatformConsent.submitFailedTitle')
    error.value = getErrorMessage(err, t('common.openPlatformConsent.submitFailed'))
    submitting.value = ''
  }
}

function redirectToURL(rawURL: string) {
  const parsed = new URL(rawURL, window.location.origin)
  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new Error('Invalid redirect URL protocol')
  }
  window.location.assign(parsed.toString())
}

function sensitivityLabel(value: Sensitivity) {
  return t(`common.openPlatformConsent.sensitivity.${value}`)
}

function sensitivityClass(value: Sensitivity) {
  return {
    'bg-secondary/10 text-secondary': value === 'low',
    'bg-warning/10 text-warning': value === 'medium',
    'bg-danger/10 text-danger': value === 'high' || value === 'very_high',
  }
}

onMounted(loadConsent)
</script>

<style scoped>
.consent-link {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--color-primary);
  font-size: 0.875rem;
  font-weight: 600;
  text-decoration: none;
}

.consent-link:hover {
  color: var(--color-primary-dark);
}

.permission-row {
  padding: 1rem;
  border-top: 1px solid var(--color-border);
}

.permission-row:first-child {
  border-top: 0;
}

.sensitivity-badge {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  border-radius: 0.375rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 700;
}
</style>
