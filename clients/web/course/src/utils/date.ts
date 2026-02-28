// 日期格式化
export const formatDate = (date: string | Date, format = 'YYYY-MM-DD'): string => {
  const d = new Date(date)
  if (isNaN(d.getTime())) {
    return '-'
  }
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hour = String(d.getHours()).padStart(2, '0')
  const minute = String(d.getMinutes()).padStart(2, '0')

  return format
    .replace('YYYY', String(year))
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hour)
    .replace('mm', minute)
}

// i18n 翻译函数类型
type TFunc = (key: string, params?: Record<string, unknown>) => string

/**
 * 相对时间格式化（i18n 感知）
 * 用于 ReviewCard、ReplyCard、NotificationItem 等组件
 */
export const formatRelativeTime = (
  date: string | Date,
  locale: string,
  t: TFunc
): string => {
  const d = date instanceof Date ? date : new Date(date)
  if (isNaN(d.getTime())) return '-'

  const diff = Date.now() - d.getTime()

  if (diff < 60_000) return t('common.time.justNow')
  if (diff < 3_600_000) return t('common.time.minutesAgo', { n: Math.floor(diff / 60_000) })
  if (diff < 86_400_000) return t('common.time.hoursAgo', { n: Math.floor(diff / 3_600_000) })
  if (diff < 604_800_000) return t('common.time.daysAgo', { n: Math.floor(diff / 86_400_000) })

  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  }).format(d)
}

/**
 * 绝对时间格式化（locale 感知）
 * 用于 LogsPage、ReportsPage 等管理后台页面
 * 使用 Intl.DateTimeFormat 保证跨浏览器一致性
 */
export const formatAbsoluteTime = (dateStr: string, locale: string): string => {
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return '-'
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(d)
}

/**
 * 本地化日期格式化
 * 基于 Intl.DateTimeFormat，自动适配 locale
 */
export const formatLocalDate = (
  date: string | Date,
  locale: string,
  options?: Intl.DateTimeFormatOptions
): string => {
  const d = date instanceof Date ? date : new Date(date)
  if (isNaN(d.getTime())) return '-'
  return new Intl.DateTimeFormat(locale, options ?? {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  }).format(d)
}

/**
 * 本地化日期时间格式化
 * 包含日期和时间，适用于详情页展示
 */
export const formatLocalDateTime = (
  date: string | Date,
  locale: string
): string => {
  const d = date instanceof Date ? date : new Date(date)
  if (isNaN(d.getTime())) return '-'
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(d)
}
