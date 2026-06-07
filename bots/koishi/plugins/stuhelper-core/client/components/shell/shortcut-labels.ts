export interface ShellShortcutLabels {
  readonly search: string
  readonly chat: string
}

function readNavigatorPlatform(): string {
  return typeof navigator === 'undefined' ? '' : navigator.platform
}

export function getShellShortcutModifier(platform = readNavigatorPlatform()): '⌘' | 'Ctrl' {
  return platform.toLowerCase().includes('mac') ? '⌘' : 'Ctrl'
}

export function getShellShortcutLabels(platform?: string): ShellShortcutLabels {
  const modifier = getShellShortcutModifier(platform)

  return {
    search: modifier === '⌘' ? '⌘K' : 'Ctrl+K',
    chat: modifier === '⌘' ? '⌘/' : 'Ctrl+/',
  }
}
