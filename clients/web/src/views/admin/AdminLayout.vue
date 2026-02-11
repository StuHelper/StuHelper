<template>
  <div class="admin-layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <h2>{{ t('admin.title') }}</h2>
      </div>
      <nav class="sidebar-nav">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
        >
          <component :is="item.icon" class="nav-icon" />
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
    </aside>
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { h, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const DashboardIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
  h('path', { d: 'M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z' })
])

const ReportIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
  h('path', { d: 'M14.4 6L14 4H5v17h2v-7h5.6l.4 2h7V6z' })
])

const ReviewIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
  h('path', { d: 'M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z' })
])

const LogIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, [
  h('path', { d: 'M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z' })
])

const navItems = computed(() => [
  { path: '/admin', label: t('admin.nav.dashboard'), icon: DashboardIcon },
  { path: '/admin/reports', label: t('admin.nav.reports'), icon: ReportIcon },
  { path: '/admin/reviews', label: t('admin.nav.reviews'), icon: ReviewIcon },
  { path: '/admin/logs', label: t('admin.nav.logs'), icon: LogIcon }
])
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
}

.sidebar {
  width: 220px;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
  flex-shrink: 0;
}

.sidebar-header {
  padding: var(--space-4);
  border-bottom: 1px solid var(--border);
}

.sidebar-header h2 {
  margin: 0;
  font-family: var(--font-sans);
  font-size: var(--text-base);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}

.sidebar-nav {
  padding: var(--space-2);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  color: var(--text-muted);
  text-decoration: none;
  font-size: var(--text-sm);
  border-radius: var(--radius-sm);
  transition: color var(--duration-fast);
}

.nav-item:hover {
  color: var(--text-primary);
}

.nav-item.router-link-active {
  color: var(--brand-primary);
  font-weight: var(--weight-medium);
  background: color-mix(in srgb, var(--brand-primary) 8%, transparent);
}

.nav-icon {
  width: 18px;
  height: 18px;
}

.main-content {
  flex: 1;
  padding: var(--space-6);
  overflow-y: auto;
}
</style>
