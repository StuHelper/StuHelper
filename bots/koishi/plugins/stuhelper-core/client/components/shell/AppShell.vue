<template>
  <div
    class="sh-shell"
    :data-rail-expanded="shell.railExpanded.value ? 'true' : 'false'"
  >
    <NavRail
      class="sh-shell__rail"
      :navigation="navigation"
      :pulse="pulse.state"
      :version="version"
    />
    <button
      v-if="mobileRailOpen"
      type="button"
      class="sh-shell__rail-scrim"
      aria-label="收起导航"
      @click="shell.closeRail()"
    ></button>
    <CommandBar
      class="sh-shell__cmd"
      :navigation="navigation"
      :pulse="pulse.state"
    />
    <main class="sh-shell__main">
      <keep-alive>
        <component :is="activeComponent" :navigation="navigation" />
      </keep-alive>
    </main>

    <EntityOverlay :navigation="navigation" />
    <ChatDock />
    <SearchPanel :navigation="navigation" />
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'

import { provideAppShell } from '../../composables/use-app-shell'
import { useConsoleNavigation } from '../../composables/use-console-navigation'
import { useConsolePages } from '../../composables/use-console-pages'
import { usePulse } from '../../composables/use-pulse'
import { consoleViewTitle } from '../../models/views'
import NavRail from './NavRail.vue'
import CommandBar from './CommandBar.vue'
import EntityOverlay from './EntityOverlay.vue'
import ChatDock from './ChatDock.vue'
import SearchPanel from './SearchPanel.vue'
import { MOBILE_RAIL_BREAKPOINT } from './breakpoints'

defineProps<{
  version: string
}>()

const shell = provideAppShell()
const route = useRoute()
const navigation = useConsoleNavigation()
const pages = useConsolePages()
const pulse = usePulse()
const baseTitle = normalizeBaseTitle(document.title)

const activeComponent = computed(() => pages.resolve(navigation.state.value.view))
const pageTitle = computed(() => `${consoleViewTitle(navigation.state.value.view)} · ${baseTitle}`)
const mobileRailOpen = computed(() =>
  navigation.viewportWidth.value <= MOBILE_RAIL_BREAKPOINT &&
  shell.railExpanded.value &&
  !shell.railPinned.value,
)

watch(
  pageTitle,
  (title) => {
    document.title = title
  },
  { immediate: true },
)

watch(
  () => route.path,
  (path) => {
    if (isStuhelperShellPath(path)) {
      syncShellLocation()
    } else {
      shell.closeTransientOverlays()
    }
  },
  { immediate: true },
)

watch(
  () => navigation.viewportWidth.value,
  (width) => {
    if (width <= MOBILE_RAIL_BREAKPOINT) {
      shell.closeRail()
    }
  },
)

function onKeydown(event: KeyboardEvent): void {
  if (event.defaultPrevented) return
  const isMac = navigator.platform.toLowerCase().includes('mac')
  const meta = isMac ? event.metaKey : event.ctrlKey
  const key = event.key.toLowerCase()
  if (meta && key === 'k') {
    event.preventDefault()
    shell.toggleSearch()
    return
  }
  if (meta && key === '/') {
    event.preventDefault()
    shell.toggleChat()
    return
  }
  if (key === 'escape') {
    if (shell.searchOpen.value) {
      shell.closeSearch()
      event.preventDefault()
    } else if (shell.entityTarget.value) {
      shell.closeEntity()
      event.preventDefault()
    } else if (shell.chatOpen.value && !shell.chatMinimized.value) {
      shell.closeChat()
      event.preventDefault()
    } else if (mobileRailOpen.value) {
      shell.closeRail()
      event.preventDefault()
    }
  }
}

function onKeydownCapture(event: KeyboardEvent): void {
  if (
    event.key.toLowerCase() !== 'escape' ||
    !mobileRailOpen.value ||
    shell.searchOpen.value ||
    shell.entityTarget.value ||
    (shell.chatOpen.value && !shell.chatMinimized.value)
  ) {
    return
  }
  shell.closeRail()
  event.preventDefault()
}

onActivated(() => {
  if (isStuhelperShellPath(route.path)) {
    syncShellLocation()
  }
})

onMounted(() => {
  window.addEventListener('keydown', onKeydownCapture, { capture: true })
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydownCapture, { capture: true })
  window.removeEventListener('keydown', onKeydown)
  shell.closeTransientOverlays()
  document.title = baseTitle
})

function isStuhelperShellPath(path: string): boolean {
  return path === '/stuhelper' || path.startsWith('/stuhelper/')
}

function syncShellLocation(): void {
  navigation.syncFromLocation()
  document.title = pageTitle.value
}

function normalizeBaseTitle(title: string): string {
  const marker = 'StuHelper 群管中心'
  const index = title.indexOf(marker)
  return index >= 0 ? title.slice(index) : title
}
</script>
