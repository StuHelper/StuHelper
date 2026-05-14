<template>
  <div class="min-h-screen relative z-0">
    <AppHeader />

    <main class="app-shell-main" :class="mainPaddingClass">
      <slot />
    </main>

    <CommandPalette />
    <Toast />
    <FloatingModuleNav />
  </div>
</template>

<script setup lang="ts">
import * as Vue from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from './AppHeader.vue'
import FloatingModuleNav from './FloatingModuleNav.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import Toast from '@/components/common/Toast.vue'

const route = useRoute()

const mainPaddingClass = Vue.computed(() => {
  if (route.path === '/course') {
    return 'pt-[var(--navbar-height)] max-tablet:pt-[var(--mobile-header-height)]'
  }
  return 'pt-[var(--navbar-height)]'
})
</script>
