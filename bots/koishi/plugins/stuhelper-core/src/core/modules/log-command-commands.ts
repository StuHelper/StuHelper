import type { Session } from 'koishi'

import type { CommandLogRecord, LogModule } from './log.module'
import { filterLogs, formatLogList, formatStats, type LogFilterOptions, type LogStatsOptions } from './log-formatters'

const DEFAULT_COMMAND_LOG_LIMIT = 10
const COMMAND_LOG_SAMPLE_MULTIPLIER = 10
const MAX_COMMAND_LOG_SAMPLE_SIZE = 1000
const DEFAULT_CLEAR_DAYS = 0
const DEFAULT_EXPORT_DAYS = 7
const DAY_MS = 24 * 60 * 60 * 1000
const JSON_INDENT_SPACES = 2

interface CommandLogCheckOptions extends LogFilterOptions {
  readonly limit?: number
}

interface CommandLogStatsOptions extends LogStatsOptions {
  readonly limit?: number
  readonly sortBy?: 'count' | 'time' | 'guild' | 'user' | string
}

interface CommandLogClearOptions {
  readonly days?: number
  readonly all?: boolean
}

interface CommandLogExportOptions {
  readonly days?: number
  readonly format?: 'json' | 'csv' | string
}

export function registerCommandLogCommands(host: LogModule): void {
  registerCheckCommand(host)
  registerStatsCommand(host)
  registerClearCommand(host)
  registerExportCommand(host)
}

function registerCheckCommand(host: LogModule): void {
  host.registerCommand({
    name: 'cmdlogs.check',
    desc: '查看命令执行日志',
    permNode: 'cmdlogs-check',
    permDesc: '查看命令执行日志',
    usage: '查看命令执行记录，支持多种过滤选项',
    examples: ['cmdlogs.check -l 20', 'cmdlogs.check -u 123456 -f'],
  })
    .alias('命令日志')
    .option('limit', '-l <number> 显示条数', { fallback: DEFAULT_COMMAND_LOG_LIMIT })
    .option('user', '-u <userId> 筛选特定用户')
    .option('command', '-c <command> 筛选特定命令')
    .option('failed', '-f 只显示失败的命令')
    .option('private', '-p 只显示私聊命令')
    .option('guild', '-g <guildId> 筛选特定群组')
    .option('platform', '--platform <platform> 筛选特定平台')
    .option('authority', '-a <level> 筛选特定权限级别')
    .option('since', '-s <date> 显示指定时间之后的日志')
    .option('until', '--until <date> 显示指定时间之前的日志')
    .action(async ({ options }) => handleCheckCommand(host, options))
}

function registerStatsCommand(host: LogModule): void {
  host.registerCommand({
    name: 'cmdlogs.stats',
    desc: '查看命令使用统计',
    permNode: 'cmdlogs-stats',
    permDesc: '查看命令统计',
    usage: '统计命令使用情况，支持过滤和排序',
  })
    .alias('命令统计')
    .option('limit', '-l <number> 显示前N个命令', { fallback: DEFAULT_COMMAND_LOG_LIMIT })
    .option('command', '-c <command> 筛选特定命令')
    .option('user', '-u <userId> 筛选特定用户')
    .option('guild', '-g <guildId> 筛选特定群组')
    .option('sortBy', '--sort <type> 排序方式: count, time, guild, user', { fallback: 'count' })
    .action(async ({ options }) => handleStatsCommand(host, options))
}

function registerClearCommand(host: LogModule): void {
  host.registerCommand({
    name: 'cmdlogs.clear',
    desc: '清除命令日志',
    permNode: 'cmdlogs-clear',
    permDesc: '清除命令日志',
    usage: '-d 天数 清除N天前，--all 清除所有',
  })
    .alias('清理日志')
    .option('days', '-d <number> 清除N天前的日志', { fallback: DEFAULT_CLEAR_DAYS })
    .option('all', '--all 清除所有日志')
    .action(async ({ session, options }) => handleClearCommand(host, session, options))
}

function registerExportCommand(host: LogModule): void {
  host.registerCommand({
    name: 'cmdlogs.export',
    desc: '导出命令日志',
    permNode: 'cmdlogs-export',
    permDesc: '导出命令日志',
    usage: '-d 天数 导出最近N天，-f json/csv 格式',
  })
    .alias('导出日志')
    .option('days', '-d <number> 导出最近N天的日志', { fallback: DEFAULT_EXPORT_DAYS })
    .option('format', '-f <format> 导出格式 (json|csv)', { fallback: 'json' })
    .action(async ({ options }) => handleExportCommand(host, options))
}

function handleCheckCommand(host: LogModule, options: CommandLogCheckOptions): string {
  try {
    const limit = options.limit ?? DEFAULT_COMMAND_LOG_LIMIT
    const sampleSize = Math.min(limit * COMMAND_LOG_SAMPLE_MULTIPLIER, MAX_COMMAND_LOG_SAMPLE_SIZE)
    const logs = host.readCommandLogs().slice(-sampleSize).reverse()
    if (logs.length === 0) return '暂无命令执行记录'

    const filteredLogs = filterLogs(logs, options).slice(0, limit)
    if (filteredLogs.length === 0) return '没有符合条件的命令记录'

    return formatLogList(filteredLogs, logs.length)
  } catch (error) {
    return `获取命令日志失败: ${errorMessage(error)}`
  }
}

function handleStatsCommand(host: LogModule, options: CommandLogStatsOptions): string {
  try {
    const allLogs = host.readCommandLogs()
    if (allLogs.length === 0) return '暂无命令使用记录'

    const filteredLogs = filterLogs(allLogs, options)
    if (filteredLogs.length === 0) return '没有符合条件的命令记录'

    return formatStats(filteredLogs, options)
  } catch (error) {
    return `获取命令统计失败: ${errorMessage(error)}`
  }
}

function handleClearCommand(host: LogModule, session: Session, options: CommandLogClearOptions): string {
  try {
    if (options.all) {
      host.saveCommandLogs([])
      host.clearCommandStats()
      void host.logCommand({ session, command: 'cmdlogs.clear', target: 'all', result: 'success' })
      return '已清除所有命令日志'
    }
    const days = options.days ?? DEFAULT_CLEAR_DAYS
    if (days <= 0) {
      return '请指定 --all 清除所有日志，或使用 -d <天数> 清除指定天数前的日志'
    }

    const removedCount = cleanOldLogs(host, days)
    void host.logCommand({
      session,
      command: 'cmdlogs.clear',
      target: `${days}days`,
      result: `removed ${removedCount}`,
    })
    return `已清理 ${removedCount} 条超过 ${days} 天的命令日志`
  } catch (error) {
    return `清理日志失败: ${errorMessage(error)}`
  }
}

function handleExportCommand(host: LogModule, options: CommandLogExportOptions): string {
  try {
    const logs = host.readCommandLogs()
    const days = options.days ?? DEFAULT_EXPORT_DAYS
    const cutoffTime = Date.now() - (days * DAY_MS)
    const filteredLogs = logs.filter(log => new Date(log.timestamp).getTime() > cutoffTime)
    if (filteredLogs.length === 0) return `最近 ${days} 天没有命令执行记录`

    if (options.format === 'csv') {
      return formatCsvExport(filteredLogs)
    }
    return `JSON格式日志 (${filteredLogs.length} 条记录)\n\n${JSON.stringify(filteredLogs, null, JSON_INDENT_SPACES)}`
  } catch (error) {
    return `导出日志失败: ${errorMessage(error)}`
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function cleanOldLogs(host: LogModule, daysToKeep: number): number {
  try {
    const cutoffTime = Date.now() - (daysToKeep * DAY_MS)
    const logs = host.readCommandLogs()
    const filteredLogs = logs.filter(log => new Date(log.timestamp).getTime() > cutoffTime)
    const removedCount = logs.length - filteredLogs.length

    host.saveCommandLogs(filteredLogs)
    host.loadStats()
    return removedCount
  } catch (error) {
    host.ctx.logger('stuhelper-core:log').error('清理命令日志失败: %o', error)
    return 0
  }
}

function formatCsvExport(logs: CommandLogRecord[]): string {
  const csvHeader = 'timestamp,userId,username,userAuthority,guildId,platform,command,success,executionTime,error\n'
  const csvRows = logs.map(log =>
    `"${log.timestamp}","${log.userId}","${log.username}","${log.userAuthority || ''}","${log.guildId || ''}","${log.platform}","${log.command}","${log.success}","${log.executionTime}","${log.error || ''}"`,
  ).join('\n')

  return `CSV格式日志 (${logs.length} 条记录)\n\n${csvHeader}${csvRows}`
}
