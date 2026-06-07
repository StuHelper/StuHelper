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
  if (typeof value !== 'string') return ''
  const message = value.trim()
  if (!message) return ''
  if (isNetworkFailureMessage(message)) {
    return 'StuHelper 平台服务暂时不可用，请检查后端地址、网络或服务令牌配置。'
  }
  return message
}

function isNetworkFailureMessage(message: string): boolean {
  return /(?:^|\b)(?:fetch failed|failed to fetch|networkerror|econnrefused|und_err_socket|socket hang up|connect econnrefused)(?:\b|$)/i.test(
    message,
  )
}

function isErrorLikeRecord(value: unknown): value is {
  readonly message?: unknown
  readonly error?: unknown
  readonly reason?: unknown
} {
  return typeof value === 'object' && value !== null
}
