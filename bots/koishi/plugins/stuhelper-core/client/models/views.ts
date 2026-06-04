export const CONSOLE_VIEW_IDS = [
  'dashboard',
  'admission',
  'config',
  'warns',
  'blacklist',
  'identity',
  'review',
  'roles',
  'logs',
  'chat',
  'subscriptions',
  'settings',
  'system',
] as const

export type ConsoleViewId = (typeof CONSOLE_VIEW_IDS)[number]

export const DEFAULT_CONSOLE_VIEW: ConsoleViewId = 'dashboard'

export function isConsoleViewId(value: string | null | undefined): value is ConsoleViewId {
  return Boolean(value) && CONSOLE_VIEW_IDS.includes(value as ConsoleViewId)
}
