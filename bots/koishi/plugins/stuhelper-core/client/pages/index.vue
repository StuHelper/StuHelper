<template>
  <k-layout class="stuhelperGroupCenter-app">
    <TopNavigation :version="pkg.version" :navigation="navigation" :items="allMenuItems" />

    <div class="main-content">
      <keep-alive>
        <component :is="activeComponent" :navigation="navigation" />
      </keep-alive>
    </div>
  </k-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import pkg from '../../package.json'
import TopNavigation from '../components/TopNavigation.vue'
import { useConsoleNavigation } from '../composables/use-console-navigation'
import { useConsolePages } from '../composables/use-console-pages'
import type { ConsoleViewId } from '../models/views'

const navigation = useConsoleNavigation()
const pages = useConsolePages()

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

const activeComponent = computed(() => pages.resolve(navigation.state.value.view))
</script>

<style scoped>
.stuhelperGroupCenter-app {
  background: var(--bg1);
  height: 100vh;
  min-height: 0;
  font-family: var(--sh-font-ui);
}

.main-content {
  max-width: 1440px;
  margin: 0 auto;
  padding: 16px;
  height: calc(100vh - 52px);
  overflow: auto;
  box-sizing: border-box;
}

@media (max-width: 767px) {
  .main-content {
    padding: 12px;
  }
}

@media (max-width: 479px) {
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
