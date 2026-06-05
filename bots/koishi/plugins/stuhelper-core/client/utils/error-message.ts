export function errorMessage(cause: unknown, fallback: string): string {
  if (cause instanceof Error) return normalizeMessage(cause.message) || fallback
  if (typeof cause === 'string') return normalizeMessage(cause) || fallback
  if (isErrorLikeRecord(cause)) {
    return normalizeMessage(cause.message)
      || normalizeMessage(cause.error)
      || normalizeMessage(cause.reason)
      || fallback
  }
  if (cause === undefined || cause === null) return fallback
  return normalizeMessage(String(cause)) || fallback
}

function normalizeMessage(value: unknown): string {
  if (value instanceof Error) return normalizeMessage(value.message)
  return typeof value === 'string' && value.trim() ? value.trim() : ''
}

function isErrorLikeRecord(value: unknown): value is {
  readonly message?: unknown
  readonly error?: unknown
  readonly reason?: unknown
} {
  return typeof value === 'object' && value !== null
}
