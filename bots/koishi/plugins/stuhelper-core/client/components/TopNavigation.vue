<template>
  <div class="top-nav">
    <div ref="containerRef" class="nav-container">
      <div ref="logoRef" class="logo-area">
        <span class="logo-text">STUHELPER GROUP CENTER</span>
        <span class="version-text">v{{ version }}</span>
      </div>

      <button class="mobile-menu-btn" @click="mobileMenuOpen = !mobileMenuOpen">
        <k-icon :name="mobileMenuOpen ? 'stuhelperGroupCenter:octicons.x' : 'stuhelperGroupCenter:octicons.three-bars'" />
      </button>

      <div ref="tabsRef" class="nav-tabs" :class="{ open: mobileMenuOpen }">
        <button
          v-for="item in visibleItems"
          :key="item.id"
          class="nav-tab"
          :class="{ active: activeView === item.id }"
          @click="handleSelectView(item.id)"
        >
          <k-icon :name="item.icon" class="tab-icon" />
          <span>{{ item.label }}</span>
        </button>

        <div v-if="overflowItems.length" ref="moreRef" class="more-menu">
          <button class="nav-tab" :class="{ active: overflowActive }" @click="moreMenuOpen = !moreMenuOpen">
            <k-icon name="stuhelperGroupCenter:octicons.chevron-down" class="tab-icon" />
            <span>More</span>
          </button>
          <div v-if="moreMenuOpen" class="more-menu__panel">
            <button
              v-for="item in overflowItems"
              :key="item.id"
              class="more-menu__item"
              @click="handleSelectView(item.id)"
            >
              {{ item.label }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="mobileMenuOpen" class="mobile-menu-overlay" @click="mobileMenuOpen = false"></div>
    </div>

    <div ref="measureRef" class="nav-measure" aria-hidden="true">
      <button
        v-for="item in items"
        :key="item.id"
        class="nav-tab"
        type="button"
      >
        <k-icon :name="item.icon" class="tab-icon" />
        <span>{{ item.label }}</span>
      </button>
      <button class="nav-tab" type="button">
        <k-icon name="stuhelperGroupCenter:octicons.chevron-down" class="tab-icon" />
        <span>More</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import type { ConsoleViewId } from '../models/views'

const NAV_GAP = 4
const NAV_RESERVED_SPACE = 8

interface MenuItem {
  readonly id: ConsoleViewId
  readonly label: string
  readonly icon: string
}

const props = defineProps<{
  readonly version: string
  readonly navigation: ConsoleNavigationController
  readonly items: readonly MenuItem[]
}>()

const mobileMenuOpen = ref(false)
const moreMenuOpen = ref(false)
const visibleCount = ref(props.items.length)
const containerRef = ref<HTMLElement | null>(null)
const logoRef = ref<HTMLElement | null>(null)
const tabsRef = ref<HTMLElement | null>(null)
const measureRef = ref<HTMLElement | null>(null)
const moreRef = ref<HTMLElement | null>(null)

const activeView = computed(() => props.navigation.state.value.view)
const visibleItems = computed(() => props.items.slice(0, visibleCount.value))
const overflowItems = computed(() => props.items.slice(visibleCount.value))
const overflowActive = computed(() => overflowItems.value.some((item) => item.id === activeView.value))

let resizeObserver: ResizeObserver | null = null

watch(
  () => props.navigation.viewportWidth.value,
  () => {
    if (!props.navigation.isCompact.value) {
      mobileMenuOpen.value = false
    }
    moreMenuOpen.value = false
    scheduleMeasurement()
  },
)

onMounted(() => {
  resizeObserver = new ResizeObserver(scheduleMeasurement)
  if (containerRef.value) resizeObserver.observe(containerRef.value)
  if (logoRef.value) resizeObserver.observe(logoRef.value)
  scheduleMeasurement()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})

function handleSelectView(id: ConsoleViewId) {
  props.navigation.selectView(id)
  mobileMenuOpen.value = false
  moreMenuOpen.value = false
}

function scheduleMeasurement() {
  void nextTick(measureVisibleItems)
}

function measureVisibleItems() {
  if (props.navigation.isCompact.value) {
    visibleCount.value = props.items.length
    return
  }

  const capacity = getTabsCapacity()
  const widths = getMeasuredTabWidths()
  const moreWidth = getMeasuredMoreWidth(widths)
  visibleCount.value = calculateVisibleCount(widths, capacity, moreWidth)
}

function getTabsCapacity() {
  const container = containerRef.value
  const logo = logoRef.value
  if (!container || !logo) return 0
  const styles = window.getComputedStyle(container)
  const gap = Number.parseFloat(styles.columnGap || styles.gap || '0') || 0
  return container.clientWidth - logo.offsetWidth - gap - NAV_RESERVED_SPACE
}

function getMeasuredTabWidths() {
  const measure = measureRef.value
  if (!measure) return []
  return Array.from(measure.querySelectorAll<HTMLElement>('.nav-tab')).map((node) => node.offsetWidth)
}

function getMeasuredMoreWidth(widths: readonly number[]) {
  return widths[props.items.length] ?? moreRef.value?.offsetWidth ?? 0
}

function calculateVisibleCount(widths: readonly number[], capacity: number, moreWidth: number) {
  const itemWidths = widths.slice(0, props.items.length)
  const allWidth = sumWithGaps(itemWidths)
  if (allWidth <= capacity) return props.items.length

  for (let count = props.items.length - 1; count >= 0; count -= 1) {
    const usedWidth = sumWithGaps(itemWidths.slice(0, count))
    const totalWidth = count > 0 ? usedWidth + NAV_GAP + moreWidth : moreWidth
    if (totalWidth <= capacity) return count
  }

  return 0
}

function sumWithGaps(widths: readonly number[]) {
  if (!widths.length) return 0
  return widths.reduce((total, width) => total + width, 0) + (widths.length - 1) * NAV_GAP
}
</script>

<style scoped src="./top-navigation.css"></style>
