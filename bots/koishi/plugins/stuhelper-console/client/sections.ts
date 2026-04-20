export const CONSOLE_SECTIONS = [
  { id: 'dashboard', label: '首页驾驶舱' },
  { id: 'enforcement', label: '处置中心' },
  { id: 'identity', label: '身份认证' },
  { id: 'policy', label: '策略中心' },
  { id: 'audit', label: '审计检索' },
] as const

export type ConsoleSectionId = (typeof CONSOLE_SECTIONS)[number]['id']

export const DEFAULT_CONSOLE_SECTION: ConsoleSectionId = 'dashboard'

export function isConsoleSectionId(value: string): value is ConsoleSectionId {
  return CONSOLE_SECTIONS.some((item) => item.id === value)
}
