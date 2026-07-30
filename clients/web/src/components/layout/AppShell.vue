<template>
  <div class="min-h-screen relative z-0">
    <a class="app-shell-skip-link" href="#main-content">
      {{ t('nav.skipToContent') }}
    </a>

    <AppHeader />

    <main
      id="main-content"
      class="app-shell-main"
      :class="mainPaddingClass"
      tabindex="-1"
    >
      <slot />
    </main>

    <CommandPalette />
    <FloatingModuleNav />
  </div>
</template>

<script setup lang="ts">
import * as Vue from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppHeader from './AppHeader.vue'
import FloatingModuleNav from './FloatingModuleNav.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'

const route = useRoute()
const { t } = useI18n()

const mainPaddingClass = Vue.computed(() => {
  if (route.path === '/courses') {
    return 'pt-[var(--navbar-height)] max-tablet:pt-[var(--mobile-header-height)]'
  }
  return 'pt-[var(--navbar-height)]'
})
</script>
