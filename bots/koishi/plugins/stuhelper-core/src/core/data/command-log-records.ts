import { redactCommandLogRecord } from './log-redaction'

export interface CommandLogRecord {
  id: string
  timestamp: string
  userId: string
  username?: string
  userAuthority?: number
  guildId?: string
  guildName?: string
  channelId?: string
  platform: string
  command: string
  args: string[]
  options: Record<string, unknown>
  success: boolean
  error?: string
  executionTime: number
  result?: string
  messageId?: string
  isPrivate: boolean
}

type RecordLike = Record<string, unknown>

export function normalizeCommandLogRecords(value: unknown): CommandLogRecord[] {
  const entries = Array.isArray(value)
    ? value
    : isRecord(value) && Array.isArray(value.logs)
      ? value.logs
      : []

  return entries
    .filter(isRecord)
    .map((entry, index) => normalizeCommandLogRecord(entry, index))
}

function normalizeCommandLogRecord(entry: RecordLike, index: number): CommandLogRecord {
  const guildId = optionalString(entry.guildId)
  const error = optionalString(entry.error)
  const result = optionalString(entry.result) ?? optionalString(entry.details)
  const command = stringValue(entry.command, 'unknown')
  const userId = stringValue(entry.userId, '')
  const timestamp = normalizeTimestamp(entry.timestamp)

  return redactCommandLogRecord({
    id: stringValue(entry.id, `${timestamp}:${command}:${userId}:${index}`),
    timestamp,
    userId,
    username: optionalString(entry.username),
    userAuthority: optionalNumber(entry.userAuthority),
    guildId,
    guildName: optionalString(entry.guildName),
    channelId: optionalString(entry.channelId),
    platform: stringValue(entry.platform, 'unknown'),
    command,
    args: Array.isArray(entry.args) ? entry.args.map((item) => String(item)) : [],
    options: isRecord(entry.options) ? entry.options : {},
    success: typeof entry.success === 'boolean' ? entry.success : !error,
    error,
    executionTime: optionalNumber(entry.executionTime) ?? 0,
    result,
    messageId: optionalString(entry.messageId),
    isPrivate: typeof entry.isPrivate === 'boolean' ? entry.isPrivate : !guildId,
  })
}

function normalizeTimestamp(value: unknown): string {
  if (typeof value === 'string' && value.trim()) {
    return value
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return new Date(value).toISOString()
  }
  return new Date(0).toISOString()
}

function stringValue(value: unknown, fallback: string): string {
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  return fallback
}

function optionalString(value: unknown): string | undefined {
  const normalized = stringValue(value, '')
  return normalized ? normalized : undefined
}

function optionalNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function isRecord(value: unknown): value is RecordLike {
  return typeof value === 'object' && value !== null
}
