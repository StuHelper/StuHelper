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
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
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
    if (!isStuhelperShellPath(path)) {
      shell.closeTransientOverlays()
    }
  },
  { immediate: true },
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
    }
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  shell.closeTransientOverlays()
  document.title = baseTitle
})

function isStuhelperShellPath(path: string): boolean {
  return path === '/stuhelper' || path.startsWith('/stuhelper/')
}

function normalizeBaseTitle(title: string): string {
  const marker = 'StuHelper 群管中心'
  const index = title.indexOf(marker)
  return index >= 0 ? title.slice(index) : title
}
</script>
