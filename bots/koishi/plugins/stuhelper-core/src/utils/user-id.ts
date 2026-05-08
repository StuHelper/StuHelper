export function parseUserId(user: string | any): string {
  if (!user) return ''
  if (typeof user === 'string') return parseUserIdString(user)

  const record = user as Record<string, unknown>
  const rawId = record.id || record.userId || record.uid
  return parseUserIdString(String(rawId || user))
}

function parseUserIdString(value: string): string {
  const raw = value.trim()
  const atMatch = raw.match(/^<at\b[^>]*\bid="([^"]+)"[^>]*\/?>$/)
  if (atMatch) return atMatch[1]

  const withoutAt = raw.replace(/^@/, '').trim()
  const [, platformUserId] = withoutAt.split(':')
  return platformUserId || withoutAt
}
