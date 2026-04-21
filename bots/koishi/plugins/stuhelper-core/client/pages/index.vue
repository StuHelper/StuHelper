<template>
  <k-layout class="stuhelperGroupCenter-app">
    <div class="top-nav">
      <div class="nav-container">
        <div class="logo-area">
          <span class="logo-text">STUHELPER GROUP CENTER</span>
          <span class="version-text">v{{ pkg.version }}</span>
        </div>

        <button class="mobile-menu-btn" @click="mobileMenuOpen = !mobileMenuOpen">
          <k-icon :name="mobileMenuOpen ? 'stuhelperGroupCenter:octicons.x' : 'stuhelperGroupCenter:octicons.three-bars'" />
        </button>

        <div class="nav-tabs" :class="{ open: mobileMenuOpen }">
          <button
            v-for="item in visibleMenuItems"
            :key="item.id"
            class="nav-tab"
            :class="{ active: navigation.state.value.view === item.id }"
            @click="handleSelectView(item.id)"
          >
            <k-icon :name="item.icon" class="tab-icon" />
            <span>{{ item.label }}</span>
          </button>

          <div v-if="overflowMenuItems.length" class="more-menu">
            <button class="nav-tab" :class="{ active: overflowActive }" @click="moreMenuOpen = !moreMenuOpen">
              <k-icon name="stuhelperGroupCenter:octicons.chevron-down" class="tab-icon" />
              <span>More</span>
            </button>
            <div v-if="moreMenuOpen" class="more-menu__panel">
              <button
                v-for="item in overflowMenuItems"
                :key="item.id"
                class="more-menu__item"
                @click="handleSelectView(item.id)"
              >
                {{ item.label }}
              </button>
            </div>
          </div>
        </div>

        <div class="mobile-menu-overlay" v-if="mobileMenuOpen" @click="mobileMenuOpen = false"></div>
      </div>
    </div>

    <div class="main-content">
      <keep-alive>
        <component :is="activeComponent" :navigation="navigation" />
      </keep-alive>
    </div>
  </k-layout>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import pkg from '../../package.json'
import { useConsoleNavigation } from '../composables/use-console-navigation'
import { useConsolePages } from '../composables/use-console-pages'
import type { ConsoleViewId } from '../models/views'

const navigation = useConsoleNavigation()
const pages = useConsolePages()
const mobileMenuOpen = ref(false)
const moreMenuOpen = ref(false)

const allMenuItems = [
  { id: 'dashboard', label: '仪表盘', icon: 'stuhelperGroupCenter:octicons.apps' },
  { id: 'config', label: '配置治理', icon: 'stuhelperGroupCenter:octicons.tools' },
  { id: 'warns', label: '警告记录', icon: 'stuhelperGroupCenter:octicons.warning' },
  { id: 'blacklist', label: '黑名单', icon: 'stuhelperGroupCenter:octicons.personadd' },
  { id: 'identity', label: '身份认证', icon: 'stuhelperGroupCenter:user' },
  { id: 'review', label: '处置中心', icon: 'stuhelperGroupCenter:shield' },
  { id: 'roles', label: '角色权限', icon: 'stuhelperGroupCenter:octicons.people' },
  { id: 'logs', label: '日志检索', icon: 'stuhelperGroupCenter:octicons.log' },
  { id: 'chat', label: '实时聊天', icon: 'stuhelperGroupCenter:octicons.discussion' },
  { id: 'subscriptions', label: '订阅管理', icon: 'stuhelperGroupCenter:octicons.sub' },
  { id: 'settings', label: '设置', icon: 'stuhelperGroupCenter:octicons.gear' },
] as const satisfies ReadonlyArray<{ id: ConsoleViewId; label: string; icon: string }>

const overflowIds = new Set<ConsoleViewId>(['chat', 'subscriptions', 'settings'])

const activeComponent = computed(() => pages.resolve(navigation.state.value.view))
const visibleMenuItems = computed(() => {
  if (navigation.isCompact.value) return allMenuItems
  if (!navigation.isOverflowMode.value) return allMenuItems
  return allMenuItems.filter((item) => !overflowIds.has(item.id))
})
const overflowMenuItems = computed(() => {
  if (navigation.isCompact.value || !navigation.isOverflowMode.value) return []
  return allMenuItems.filter((item) => overflowIds.has(item.id))
})
const overflowActive = computed(() => overflowMenuItems.value.some((item) => item.id === navigation.state.value.view))

watch(
  () => navigation.viewportWidth.value,
  () => {
    if (!navigation.isOverflowMode.value) {
      moreMenuOpen.value = false
    }
    if (!navigation.isCompact.value) {
      mobileMenuOpen.value = false
    }
  },
)

function handleSelectView(id: ConsoleViewId) {
  navigation.selectView(id)
  mobileMenuOpen.value = false
  moreMenuOpen.value = false
}
</script>

<style scoped>
.stuhelperGroupCenter-app {
  background: var(--bg1);
  height: 100vh;
  min-height: 0;
  font-family: var(--gh-font-sans, -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', system-ui, sans-serif);
}

.top-nav {
  position: sticky;
  top: 0;
  z-index: 10;
  background: var(--k-card-bg);
  border-bottom: 1px solid var(--k-color-divider);
  height: 52px;
}

.nav-container {
  max-width: 1440px;
  margin: 0 auto;
  padding: 0 16px;
  height: 52px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-area {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.logo-text {
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.3px;
  color: var(--fg1);
  text-transform: uppercase;
}

.version-text {
  font-size: 10px;
  font-family: var(--gh-font-mono, 'JetBrains Mono', 'SF Mono', Consolas, monospace);
  color: var(--fg3);
  background: var(--bg3);
  padding: 1px 6px;
  border-radius: 999px;
  border: 1px solid var(--k-color-divider);
}

.nav-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}

.nav-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--fg3);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.12s ease, background-color 0.12s ease, border-color 0.12s ease;
}

.nav-tab:hover {
  color: var(--fg1);
  background: var(--bg3);
}

.nav-tab.active {
  color: var(--fg1);
  background: var(--bg3);
  border-color: var(--k-color-divider);
}

.tab-icon {
  font-size: 14px;
  width: 14px;
  height: 14px;
}

.more-menu {
  position: relative;
}

.more-menu__panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 160px;
  padding: 8px;
  border: 1px solid var(--k-color-divider);
  border-radius: 12px;
  background: var(--k-card-bg);
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.18);
}

.more-menu__item {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 0 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--fg2);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}

.more-menu__item:hover {
  background: var(--bg3);
  color: var(--fg1);
}

.main-content {
  max-width: 1440px;
  margin: 0 auto;
  padding: 16px;
  height: calc(100vh - 52px);
  overflow: auto;
  box-sizing: border-box;
}

.mobile-menu-btn {
  display: none;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--fg2);
  cursor: pointer;
}

.mobile-menu-btn:hover {
  background: var(--bg3);
}

.mobile-menu-overlay {
  display: none;
}

@media (max-width: 959px) {
  .mobile-menu-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-left: auto;
  }

  .nav-tabs {
    position: fixed;
    top: 52px;
    right: -280px;
    width: 260px;
    height: calc(100vh - 52px);
    padding: 10px;
    flex-direction: column;
    align-items: stretch;
    background: var(--k-card-bg);
    border-left: 1px solid var(--k-color-divider);
    transition: right 0.22s ease;
    z-index: 100;
  }

  .nav-tabs.open {
    right: 0;
  }

  .nav-tab {
    justify-content: flex-start;
    width: 100%;
  }

  .more-menu {
    width: 100%;
  }

  .more-menu__panel {
    position: static;
    width: 100%;
    box-shadow: none;
  }

  .mobile-menu-overlay {
    display: block;
    position: fixed;
    top: 52px;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.45);
    z-index: 99;
  }
}

@media (max-width: 767px) {
  .nav-container {
    padding: 0 12px;
  }

  .logo-text {
    font-size: 12px;
  }

  .main-content {
    padding: 12px;
  }
}

@media (max-width: 479px) {
  .version-text {
    display: none;
  }

  .main-content {
    padding: 8px;
  }
}
</style>

<style>
.stuhelperGroupCenter-app .layout-header {
  display: none !important;
}

::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background-color: var(--k-color-border);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background-color: var(--fg3);
}
</style>
