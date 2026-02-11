<template>
  <nav class="module-tabs">
    <router-link
      v-for="tab in tabs"
      :key="tab.to"
      :to="tab.to"
      class="module-tab"
      :class="{ active: isActive(tab) }"
    >
      {{ tab.label }}
    </router-link>
    <div class="tab-indicator" :style="indicatorStyle" />
  </nav>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const { t } = useI18n()

const tabs = computed(() => [
  { to: '/review', label: t('nav.review'), match: ['review', 'review-course-detail'] },
  { to: '/teacher', label: t('nav.teacher'), match: ['teacher-hub', 'teacher-profile'] },
  { to: '/spoc', label: t('nav.spoc'), match: ['spoc'] },
  { to: '/resource', label: t('nav.resource'), match: ['resource'] }
])

const indicatorStyle = ref<Record<string, string>>({})

function isActive(tab: { to: string; match: string[] }) {
  const name = route.name as string
  return tab.match.includes(name)
}

function updateIndicator() {
  const activeEl = document.querySelector('.module-tab.active') as HTMLElement
  if (!activeEl) {
    indicatorStyle.value = { opacity: '0' }
    return
  }
  const nav = activeEl.parentElement
  if (!nav) return

  const navRect = nav.getBoundingClientRect()
  const elRect = activeEl.getBoundingClientRect()

  indicatorStyle.value = {
    left: `${elRect.left - navRect.left}px`,
    width: `${elRect.width}px`,
    opacity: '1'
  }
}

watch(() => route.path, () => {
  nextTick(updateIndicator)
})

onMounted(() => {
  nextTick(updateIndicator)
})
</script>

<style scoped>
.module-tabs {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  position: relative;
}

.module-tab {
  padding: var(--space-1-5) var(--space-3);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  color: var(--text-muted);
  border-radius: var(--radius-full);
  transition: color var(--duration-base) var(--ease-smooth),
    background var(--duration-base) var(--ease-smooth);
  text-decoration: none;
  position: relative;
  z-index: 1;
  white-space: nowrap;
}

.module-tab:hover {
  color: var(--text-primary);
}

.module-tab.active {
  color: var(--text-primary);
}

.tab-indicator {
  position: absolute;
  bottom: 0;
  height: 100%;
  background: var(--bg-hover);
  border-radius: var(--radius-full);
  transition: left var(--duration-slow) var(--ease-out),
    width var(--duration-slow) var(--ease-out),
    opacity var(--duration-base) var(--ease-smooth);
  z-index: 0;
}
</style>
