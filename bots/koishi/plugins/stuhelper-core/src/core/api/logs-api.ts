import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertConsoleGuildAccess } from './console-guild-scope'
import { readCommandLogs } from './log-module-lookup'
import { filterLogs } from './scope-filters'
import type { CommandLogRecord } from '../data/command-log-records'
import { redactCommandLogRecord } from '../data/log-redaction'

const DEFAULT_LOG_PAGE = 1
const DEFAULT_LOG_PAGE_SIZE = 20
const MAX_LOG_PAGE_SIZE = 100

export function registerLogsAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/logs/search', async function (params: LogSearchParams) {
    return handleLogSearch(api, this, params)
  })
}

interface LogSearchParams {
  readonly startTime?: string | number
  readonly endTime?: string | number
  readonly command?: string
  readonly userId?: string
  readonly username?: string
  readonly details?: string
  readonly guildId?: string
  readonly page?: number
  readonly pageSize?: number
}

interface NormalizedLogSearchParams {
  readonly startTime?: number
  readonly endTime?: number
  readonly command?: string
  readonly userId?: string
  readonly username?: string
  readonly details?: string
  readonly guildId?: string
  readonly page: number
  readonly pageSize: number
}

async function handleLogSearch(api: WebSocketAPIContext, client: unknown, params: LogSearchParams) {
  try {
    const normalizedParams = normalizeLogSearchParams(params)
    const scope = await api.resolveConsoleScope(client)
    if (normalizedParams.guildId) {
      assertConsoleGuildAccess(scope, normalizedParams.guildId, 'log search')
    }

    const filteredLogs = filterLogs(await readCommandLogs(api), scope)
      .map(redactCommandLogRecord)
      .filter((log) => matchesLogSearch(log, normalizedParams))
    return success(paginateLogs(filteredLogs, normalizedParams))
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '检索日志失败')
  }
}

function matchesLogSearch(log: CommandLogRecord, params: NormalizedLogSearchParams) {
  const time = new Date(log.timestamp).getTime()
  if (params.startTime !== undefined && time < params.startTime) return false
  if (params.endTime !== undefined && time > params.endTime) return false
  if (params.command && !includesInsensitive(log.command, params.command)) return false
  if (params.userId && String(log.userId) !== params.userId) return false
  if (params.username && !includesInsensitive(log.username, params.username)) return false
  if (params.details && !matchesLogDetails(log, params.details)) return false
  if (params.guildId && String(log.guildId) !== params.guildId) return false
  return true
}

function includesInsensitive(value: unknown, keyword: string) {
  return String(value || '').toLowerCase().includes(keyword.toLowerCase())
}

function matchesLogDetails(log: CommandLogRecord, details: string) {
  const keyword = details.toLowerCase()
  const matchResult = String(log.result || '').toLowerCase().includes(keyword)
  const matchError = String(log.error || '').toLowerCase().includes(keyword)
  const matchArgs = log.args.some((arg) => String(arg).toLowerCase().includes(keyword))
  const matchOptions = JSON.stringify(log.options || {}).toLowerCase().includes(keyword)
  return matchResult || matchError || matchArgs || matchOptions
}

function paginateLogs(logs: CommandLogRecord[], params: NormalizedLogSearchParams) {
  const { page, pageSize } = params
  return {
    list: logs.slice((page - 1) * pageSize, page * pageSize),
    total: logs.length,
    page,
    pageSize,
  }
}

function normalizeLogSearchParams(input: unknown): NormalizedLogSearchParams {
  const params = isRecord(input) ? input : {}
  return {
    startTime: readOptionalTimestamp(params.startTime),
    endTime: readOptionalTimestamp(params.endTime),
    command: readOptionalString(params.command),
    userId: readOptionalString(params.userId),
    username: readOptionalString(params.username),
    details: readOptionalString(params.details),
    guildId: readOptionalString(params.guildId),
    page: readPositiveInteger(params.page, DEFAULT_LOG_PAGE),
    pageSize: readPositiveInteger(params.pageSize, DEFAULT_LOG_PAGE_SIZE, MAX_LOG_PAGE_SIZE),
  }
}

function readOptionalTimestamp(value: unknown) {
  if (value === undefined || value === null || value === '') return undefined
  const timestamp = new Date(value as string | number).getTime()
  return Number.isFinite(timestamp) ? timestamp : undefined
}

function readOptionalString(value: unknown) {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed || undefined
}

function readPositiveInteger(value: unknown, fallback: number, max?: number) {
  const parsed = typeof value === 'number'
    ? value
    : typeof value === 'string' && value.trim()
      ? Number(value)
      : NaN
  if (!Number.isFinite(parsed) || parsed < 1) {
    return fallback
  }
  const integer = Math.floor(parsed)
  return max === undefined ? integer : Math.min(integer, max)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
