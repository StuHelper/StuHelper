import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertConsoleGuildAccess } from './console-guild-scope'
import { readCommandLogs } from './log-module-lookup'
import { filterLogs } from './scope-filters'
import type { CommandLogRecord } from '../modules/log.module'

const DEFAULT_LOG_PAGE = 1
const DEFAULT_LOG_PAGE_SIZE = 20

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

async function handleLogSearch(api: WebSocketAPIContext, client: unknown, params: LogSearchParams) {
  try {
    const scope = await api.resolveConsoleScope(client)
    if (params.guildId) {
      assertConsoleGuildAccess(scope, params.guildId, 'log search')
    }

    const filteredLogs = filterLogs(await readCommandLogs(api), scope)
      .filter((log) => matchesLogSearch(log, params))
    return success(paginateLogs(filteredLogs, params))
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '检索日志失败')
  }
}

function matchesLogSearch(log: CommandLogRecord, params: LogSearchParams) {
  const time = new Date(log.timestamp).getTime()
  if (params.startTime && time < new Date(params.startTime).getTime()) return false
  if (params.endTime && time > new Date(params.endTime).getTime()) return false
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

function paginateLogs(logs: CommandLogRecord[], params: LogSearchParams) {
  const page = params.page || DEFAULT_LOG_PAGE
  const pageSize = params.pageSize || DEFAULT_LOG_PAGE_SIZE
  return {
    list: logs.slice((page - 1) * pageSize, page * pageSize),
    total: logs.length,
    page,
    pageSize,
  }
}
