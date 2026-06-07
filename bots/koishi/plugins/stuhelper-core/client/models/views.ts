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

export const CONSOLE_VIEW_TITLES: Readonly<Record<ConsoleViewId, string>> = {
  dashboard: '总览',
  admission: '入群认证',
  config: '群组配置',
  warns: '警告记录',
  blacklist: '黑名单',
  identity: '限制中',
  review: '处置中心',
  roles: '角色权限',
  logs: '日志检索',
  chat: '实时聊天',
  subscriptions: '推送订阅',
  settings: '全局设置',
  system: '系统 / 缓存',
}

export function isConsoleViewId(value: string | null | undefined): value is ConsoleViewId {
  return Boolean(value) && CONSOLE_VIEW_IDS.includes(value as ConsoleViewId)
}

export function consoleViewTitle(view: ConsoleViewId): string {
  return CONSOLE_VIEW_TITLES[view] ?? CONSOLE_VIEW_TITLES[DEFAULT_CONSOLE_VIEW]
}
