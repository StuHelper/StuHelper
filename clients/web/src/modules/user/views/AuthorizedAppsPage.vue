<template>
  <main class="mx-auto max-w-[960px] p-6 animate-fade-in max-sm:p-4">
    <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <p class="m-0 text-xs font-semibold uppercase text-primary">
          {{ t('user.identityHome.eyebrow') }}
        </p>
        <h1 class="m-0 mt-2 text-2xl font-bold text-text-primary">
          {{ t('user.authorizedApps.pageTitle') }}
        </h1>
        <p class="m-0 mt-2 max-w-[680px] text-sm leading-relaxed text-text-secondary">
          {{ t('user.authorizedApps.pageSubtitle') }}
        </p>
      </div>

      <router-link
        to="/identity"
        class="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-3 text-sm font-medium text-text-secondary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
      >
        <ArrowLeft class="size-4" aria-hidden="true" />
        {{ t('user.authorizedApps.backToIdentity') }}
      </router-link>
    </header>

    <AuthorizedAppsTab />
  </main>
</template>

<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import AuthorizedAppsTab from './AuthorizedAppsTab.vue'

const { t } = useI18n()
</script>
