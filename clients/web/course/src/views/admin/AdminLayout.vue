<template>
  <div class="flex min-h-screen">
    <aside class="w-[220px] shrink-0 bg-bg-card border-r border-border">
      <div class="p-4 border-b border-border">
        <h2 class="m-0 font-sans text-base font-extrabold tracking-tight text-text-primary">{{ t('admin.title') }}</h2>
      </div>
      <nav class="p-2">
        <router-link
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="flex items-center gap-3 px-3 py-2 text-text-muted no-underline text-sm rounded-sm transition-colors duration-fast hover:text-text-primary [&.router-link-active]:text-primary [&.router-link-active]:font-medium [&.router-link-active]:bg-primary/[0.08]"
        >
          <component :is="item.icon" class="size-[18px]" />
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
    </aside>
    <main class="flex-1 p-6 overflow-y-auto">
      <ErrorBoundary>
        <router-view />
      </ErrorBoundary>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { LayoutDashboard, Flag, MessageSquare, FileText } from 'lucide-vue-next'
import ErrorBoundary from '@/components/common/ErrorBoundary.vue'

const { t } = useI18n()

const navItems = computed(() => [
  { name: 'admin-dashboard', label: t('admin.nav.dashboard'), icon: LayoutDashboard },
  { name: 'admin-reports', label: t('admin.nav.reports'), icon: Flag },
  { name: 'admin-reviews', label: t('admin.nav.reviews'), icon: MessageSquare },
  { name: 'admin-logs', label: t('admin.nav.logs'), icon: FileText }
])
</script>
