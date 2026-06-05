interface UserIdLike {
  readonly id?: unknown
  readonly userId?: unknown
  readonly uid?: unknown
}

export function parseUserId(user: unknown): string {
  if (!user) return ''
  if (typeof user === 'string') return parseUserIdString(user)
  if (!isUserIdLike(user)) return ''

  const rawId = user.id ?? user.userId ?? user.uid
  if (rawId === null || rawId === undefined) return ''
  return parseUserIdString(String(rawId))
}

function parseUserIdString(value: string): string {
  const raw = value.trim()
  const atMatch = raw.match(/^<at\b[^>]*\bid="([^"]+)"[^>]*\/?>$/)
  if (atMatch) return atMatch[1]

  const withoutAt = raw.replace(/^@/, '').trim()
  const [, platformUserId] = withoutAt.split(':')
  return platformUserId || withoutAt
}

function isUserIdLike(value: unknown): value is UserIdLike {
  return typeof value === 'object' && value !== null
}
