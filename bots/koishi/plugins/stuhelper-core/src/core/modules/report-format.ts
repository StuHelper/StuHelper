const SECONDS_PER_MINUTE = 60
const SECONDS_PER_HOUR = 60 * SECONDS_PER_MINUTE
const SECONDS_PER_DAY = 24 * SECONDS_PER_HOUR

export function formatDurationSeconds(seconds: number): string {
  if (seconds < SECONDS_PER_MINUTE) return `${seconds}s`
  if (seconds < SECONDS_PER_HOUR) return `${Math.floor(seconds / SECONDS_PER_MINUTE)}m`
  if (seconds < SECONDS_PER_DAY) return `${Math.floor(seconds / SECONDS_PER_HOUR)}h`
  return `${Math.floor(seconds / SECONDS_PER_DAY)}d`
}

export function shorten(value: string, maxLength: number): string {
  return value.length > maxLength ? `${value.substring(0, maxLength)}...` : value
}
