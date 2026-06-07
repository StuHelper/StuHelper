import { inject, provide, ref, type InjectionKey, type Ref } from 'vue'

export type EntityKind = 'user' | 'guild'

export interface EntityRef {
  readonly kind: EntityKind
  readonly id: string
  readonly name?: string
  /** When kind='user', the guild context the chip was clicked from (if any). */
  readonly guildId?: string
}

export interface AppShellController {
  readonly railExpanded: Ref<boolean>
  readonly railPinned: Ref<boolean>
  toggleRail(): void
  closeRail(): void
  pinRail(value: boolean): void

  readonly chatOpen: Ref<boolean>
  readonly chatMinimized: Ref<boolean>
  openChat(): void
  closeChat(): void
  toggleChat(): void
  toggleChatMinimized(): void

  readonly entityTarget: Ref<EntityRef | null>
  openEntity(target: EntityRef): void
  closeEntity(): void

  readonly searchOpen: Ref<boolean>
  openSearch(): void
  closeSearch(): void
  toggleSearch(): void
  closeTransientOverlays(): void
}

const KEY: InjectionKey<AppShellController> = Symbol('stuhelper:app-shell')

export function createAppShellController(): AppShellController {
  const railPinned = ref(false)
  const railExpanded = ref(false)
  const chatOpen = ref(false)
  const chatMinimized = ref(false)
  const entityTarget = ref<EntityRef | null>(null)
  const searchOpen = ref(false)

  function toggleRail() {
    railExpanded.value = !railExpanded.value
  }

  function closeRail() {
    if (!railPinned.value) {
      railExpanded.value = false
    }
  }

  function pinRail(value: boolean) {
    railPinned.value = value
    railExpanded.value = value
  }

  function openChat() {
    entityTarget.value = null
    chatOpen.value = true
    chatMinimized.value = false
  }

  function closeChat() {
    chatOpen.value = false
    chatMinimized.value = false
  }

  function toggleChat() {
    if (chatOpen.value && !chatMinimized.value) {
      closeChat()
    } else {
      openChat()
    }
  }

  function toggleChatMinimized() {
    chatMinimized.value = !chatMinimized.value
  }

  function openEntity(target: EntityRef) {
    closeChat()
    entityTarget.value = target
  }

  function closeEntity() {
    entityTarget.value = null
  }

  function openSearch() {
    searchOpen.value = true
  }

  function closeSearch() {
    searchOpen.value = false
  }

  function toggleSearch() {
    searchOpen.value = !searchOpen.value
  }

  function closeTransientOverlays() {
    closeSearch()
    closeEntity()
    closeChat()
  }

  return {
    railExpanded,
    railPinned,
    toggleRail,
    closeRail,
    pinRail,
    chatOpen,
    chatMinimized,
    openChat,
    closeChat,
    toggleChat,
    toggleChatMinimized,
    entityTarget,
    openEntity,
    closeEntity,
    searchOpen,
    openSearch,
    closeSearch,
    toggleSearch,
    closeTransientOverlays,
  }
}

export function provideAppShell(): AppShellController {
  const controller = createAppShellController()
  provide(KEY, controller)
  return controller
}

export function useAppShell(): AppShellController {
  const controller = inject(KEY)
  if (!controller) {
    throw new Error('useAppShell() must be called inside provideAppShell()')
  }
  return controller
}
