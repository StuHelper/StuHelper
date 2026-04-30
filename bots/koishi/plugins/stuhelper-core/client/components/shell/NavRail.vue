<template>
  <aside
    class="sh-rail"
    @mouseenter="onHoverIn"
    @mouseleave="onHoverOut"
  >
    <button
      type="button"
      class="sh-rail__brand"
      :title="brandTitle"
      @click="onBrandClick"
    >
      <span class="sh-rail__brand-mark">SH</span>
      <span class="sh-rail__brand-name">
        StuHelper
        <span class="sh-rail__brand-version">v{{ version }}</span>
      </span>
    </button>

    <nav class="sh-rail__nav" aria-label="主导航">
      <section
        v-for="section in NAV_SECTIONS"
        :key="section.id"
        class="sh-rail__section"
      >
        <div class="sh-rail__section-label">{{ section.label }}</div>
        <button
          v-for="item in section.items"
          :key="item.view"
          type="button"
          class="sh-rail__item"
          :class="{ 'sh-rail__item--active': activeView === item.view }"
          :title="item.label"
          @click="select(item.view)"
        >
          <span class="sh-rail__item-icon">
            <k-icon :name="item.icon" />
          </span>
          <span class="sh-rail__item-label">{{ item.label }}</span>
          <span
            v-if="badgeFor(item.view) > 0"
            class="sh-rail__item-badge"
            :data-tone="badgeToneFor(item.view)"
          >
            {{ badgeFor(item.view) }}
          </span>
        </button>
      </section>
    </nav>

    <footer class="sh-rail__foot">
      <button
        type="button"
        class="sh-rail__pin"
        :title="shell.railPinned.value ? '收起侧栏 (取消固定)' : '固定展开侧栏'"
        @click="togglePin"
      >
        <span class="sh-rail__pin-icon">
          <k-icon :name="shell.railPinned.value ? 'stuhelperGroupCenter:octicons.x' : 'stuhelperGroupCenter:octicons.three-bars'" />
        </span>
        <span class="sh-rail__pin-label">
          {{ shell.railPinned.value ? '取消固定' : '固定展开' }}
        </span>
      </button>
    </footer>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import { useAppShell } from '../../composables/use-app-shell'
import type { ConsoleNavigationController } from '../../composables/use-console-navigation'
import type { ConsoleViewId } from '../../models/views'
import type { PulseState } from '../../composables/use-pulse'

const props = defineProps<{
  navigation: ConsoleNavigationController
  pulse: PulseState
  version: string
}>()

const shell = useAppShell()

interface NavItem {
  readonly view: ConsoleViewId
  readonly label: string
  readonly icon: string
}

interface NavSection {
  readonly id: string
  readonly label: string
  readonly items: readonly NavItem[]
}

const NAV_SECTIONS: readonly NavSection[] = [
  {
    id: 'workspace',
    label: '工作台',
    items: [
      { view: 'dashboard', label: '总览', icon: 'stuhelperGroupCenter:octicons.apps' },
      { view: 'review', label: '处置中心', icon: 'stuhelperGroupCenter:shield' },
    ],
  },
  {
    id: 'members',
    label: '成员',
    items: [
      { view: 'identity', label: '限制中', icon: 'stuhelperGroupCenter:user' },
      { view: 'warns', label: '警告记录', icon: 'stuhelperGroupCenter:octicons.warning' },
      { view: 'blacklist', label: '黑名单', icon: 'stuhelperGroupCenter:octicons.x' },
    ],
  },
  {
    id: 'config',
    label: '配置',
    items: [
      { view: 'config', label: '群组配置', icon: 'stuhelperGroupCenter:octicons.tools' },
      { view: 'roles', label: '角色权限', icon: 'stuhelperGroupCenter:octicons.people' },
      { view: 'settings', label: '全局设置', icon: 'stuhelperGroupCenter:octicons.gear' },
    ],
  },
  {
    id: 'tools',
    label: '工具',
    items: [
      { view: 'logs', label: '日志检索', icon: 'stuhelperGroupCenter:octicons.log' },
      { view: 'subscriptions', label: '推送订阅', icon: 'stuhelperGroupCenter:octicons.sub' },
      { view: 'system', label: '系统 / 缓存', icon: 'stuhelperGroupCenter:octicons.graph' },
    ],
  },
]

const activeView = computed(() => props.navigation.state.value.view)

const brandTitle = computed(() =>
  shell.railPinned.value ? 'StuHelper 群管中心 (已固定)' : 'StuHelper 群管中心',
)

function select(view: ConsoleViewId): void {
  props.navigation.selectView(view)
}

function badgeFor(view: ConsoleViewId): number {
  if (view === 'review') return props.pulse.pendingReviews
  if (view === 'identity') return props.pulse.pendingAdmissions
  return 0
}

function badgeToneFor(view: ConsoleViewId): 'danger' | 'warning' | 'primary' {
  const value = badgeFor(view)
  if (view === 'review') {
    if (value > 10) return 'danger'
    if (value > 0) return 'warning'
    return 'primary'
  }
  if (view === 'identity') {
    if (value > 20) return 'danger'
    return 'warning'
  }
  return 'primary'
}

function onHoverIn(): void {
  if (!shell.railPinned.value) {
    shell.railExpanded.value = true
  }
}

function onHoverOut(): void {
  if (!shell.railPinned.value) {
    shell.railExpanded.value = false
  }
}

function onBrandClick(): void {
  shell.toggleRail()
}

function togglePin(): void {
  shell.pinRail(!shell.railPinned.value)
}
</script>
