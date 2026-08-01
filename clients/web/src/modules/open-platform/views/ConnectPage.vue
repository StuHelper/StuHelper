<template>
  <section class="mx-auto max-w-[960px] p-6 animate-fade-in max-sm:p-4">
    <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <p class="m-0 text-xs font-semibold uppercase text-primary">
          {{ t('developer.connect.eyebrow') }}
        </p>
        <h1 class="m-0 mt-2 text-2xl font-bold text-text-primary">
          {{ t('developer.connect.title') }}
        </h1>
        <p class="m-0 mt-2 max-w-[720px] text-sm leading-relaxed text-text-secondary">
          {{ t('developer.connect.subtitle', { issuer: identityIssuer }) }}
        </p>
      </div>

      <div class="flex shrink-0 flex-wrap gap-2">
        <router-link
          to="/identity"
          class="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-3 text-sm font-medium text-text-secondary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
        >
          <ArrowLeft class="size-4" aria-hidden="true" />
          {{ t('developer.connect.backToIdentity') }}
        </router-link>
        <router-link
          to="/developers/apps"
          class="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-text-primary px-3 text-sm font-semibold text-bg-base no-underline transition-colors duration-fast hover:bg-primary hover:text-white"
        >
          <KeyRound class="size-4" aria-hidden="true" />
          {{ t('developer.connect.developerApps') }}
        </router-link>
      </div>
    </header>

    <ConnectEndpointsPanel />
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowLeft, KeyRound } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import ConnectEndpointsPanel from '../components/ConnectEndpointsPanel.vue'
import { normalizeSsoIssuer } from '../connectEndpoints'

const { t } = useI18n()

const identityIssuer = computed(() =>
  normalizeSsoIssuer(
    import.meta.env.VITE_SSO_URL,
    typeof window === 'undefined' ? undefined : window.location.origin,
  ),
)
</script>
