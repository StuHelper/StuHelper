import { onMounted, onUnmounted, ref } from 'vue'

import {
  parseConsoleSearch,
  updateConsoleUrl,
  type ConsoleSearchState,
} from './navigation'

export interface RouteUpdateOptions {
  history?: 'push' | 'replace'
}

export function useConsoleNavigation() {
  const routeState = ref<ConsoleSearchState>(
    parseConsoleSearch(new URLSearchParams(window.location.search)),
  )

  function restoreRouteState() {
    routeState.value = parseConsoleSearch(new URLSearchParams(window.location.search))
  }

  onMounted(() => {
    window.addEventListener('popstate', restoreRouteState)
  })

  onUnmounted(() => {
    window.removeEventListener('popstate', restoreRouteState)
  })

  function setRouteState(
    next: Partial<ConsoleSearchState>,
    options: RouteUpdateOptions = {},
  ) {
    routeState.value = { ...routeState.value, ...next }
    const url = updateConsoleUrl(new URL(window.location.href), routeState.value)
    if (url.href !== window.location.href) {
      const method = options.history === 'replace' ? 'replaceState' : 'pushState'
      window.history[method](window.history.state, '', url)
    }
  }

  function getSelectedQueueId(section: ConsoleSearchState['section'], queue: string) {
    if (routeState.value.section !== section) return ''
    if (routeState.value.queue && routeState.value.queue !== queue) return ''
    return routeState.value.id
  }

  function selectQueueItem(
    section: ConsoleSearchState['section'],
    queue: string,
    id: string,
  ) {
    setRouteState({ section, queue, id, source: 'direct' })
  }

  return {
    routeState,
    setRouteState,
    getSelectedQueueId,
    selectQueueItem,
    restoreRouteState,
  }
}
