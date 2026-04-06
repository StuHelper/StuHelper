import { translate } from '@/i18n'

export function formatDateTime(value?: string | null): string {
  if (!value) return '--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(
    date.getDate()
  ).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(
    date.getMinutes()
  ).padStart(2, '0')}`
}

export function formatShortDate(value?: string | null): string {
  if (!value) return '--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return translate('common.shortDate', {
    month: date.getMonth() + 1,
    day: date.getDate(),
  })
}

export function averageRating(ratings: Record<string, number> | undefined): string {
  if (!ratings) return '--'
  const values = Object.values(ratings).filter((value) => typeof value === 'number')
  if (values.length === 0) return '--'
  const avg = values.reduce((sum, value) => sum + value, 0) / values.length
  return avg.toFixed(1)
}

export function toPercent(part: number, total: number): string {
  if (!total) return '0%'
  return `${Math.round((part / total) * 100)}%`
}

export function truncateText(value: string | null | undefined, maxLength: number): string {
  const trimmed = value?.trim() ?? ''
  if (!trimmed) return ''
  if (trimmed.length <= maxLength) return trimmed
  return `${trimmed.slice(0, maxLength)}…`
}
