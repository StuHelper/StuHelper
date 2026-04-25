import { computed, onBeforeUnmount, onMounted, ref, type ComputedRef, type Ref } from 'vue'

import {
  createConsoleQuery,
  mergeConsoleQuery,
  parseConsoleQuery,
  type ConsoleNavigationState,
} from '../models/navigation'
import type { ConsoleViewId } from '../models/views'

export interface ConsoleNavigationController {
  state: Ref<ConsoleNavigationState>
  viewportWidth: Ref<number>
  selectView: (view: ConsoleViewId, context?: Partial<ConsoleNavigationState>) => void
  replaceState: (patch: Partial<ConsoleNavigationState>) => void
  pushState: (patch: Partial<ConsoleNavigationState>) => void
  jumpTo: (target: Partial<ConsoleNavigationState> & { view: ConsoleViewId }) => void
  isCompact: ComputedRef<boolean>
  isOverflowMode: ComputedRef<boolean>
}

const EMPTY_CONTEXT: Omit<ConsoleNavigationState, 'view'> = {
  workspace: null,
  guildId: null,
  memberId: null,
  itemId: null,
  tab: null,
  keyword: '',
}

export function useConsoleNavigation(win = window): ConsoleNavigationController {
  const state = ref(parseConsoleQuery(new URLSearchParams(win.location.search)))
  const viewportWidth = ref(win.innerWidth)

  const syncFromLocation = () => {
    state.value = parseConsoleQuery(new URLSearchParams(win.location.search))
  }

  const syncViewportWidth = () => {
    viewportWidth.value = win.innerWidth
  }

  const updateHistory = (
    nextState: ConsoleNavigationState,
    method: 'pushState' | 'replaceState',
  ) => {
    state.value = nextState
    const nextUrl = mergeConsoleQuery(new URL(win.location.href), nextState)
    win.history[method]({}, '', `${nextUrl.pathname}${nextUrl.search}${nextUrl.hash}`)
  }

  const pushState = (patch: Partial<ConsoleNavigationState>) => {
    updateHistory({ ...state.value, ...patch }, 'pushState')
  }

  const replaceState = (patch: Partial<ConsoleNavigationState>) => {
    updateHistory({ ...state.value, ...patch }, 'replaceState')
  }

  const selectView = (view: ConsoleViewId, context: Partial<ConsoleNavigationState> = {}) => {
    updateHistory({ view, ...EMPTY_CONTEXT, ...context }, 'pushState')
  }

  const jumpTo = (target: Partial<ConsoleNavigationState> & { view: ConsoleViewId }) => {
    updateHistory({
      ...state.value,
      ...EMPTY_CONTEXT,
      ...target,
    }, 'pushState')
  }

  onMounted(() => {
    win.addEventListener('popstate', syncFromLocation)
    win.addEventListener('resize', syncViewportWidth)

    const normalized = createConsoleQuery(state.value).toString()
    const current = new URLSearchParams(win.location.search).toString()
    if (normalized !== current) {
      updateHistory(state.value, 'replaceState')
    }
  })

  onBeforeUnmount(() => {
    win.removeEventListener('popstate', syncFromLocation)
    win.removeEventListener('resize', syncViewportWidth)
  })

  return {
    state,
    viewportWidth,
    selectView,
    replaceState,
    pushState,
    jumpTo,
    isCompact: computed(() => viewportWidth.value < 960),
    isOverflowMode: computed(() => viewportWidth.value >= 960 && viewportWidth.value < 1500),
  }
}
