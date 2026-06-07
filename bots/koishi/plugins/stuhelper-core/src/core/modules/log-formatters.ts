import type { CommandLogRecord } from './log.module'
import { redactCommandLogRecord } from './log-redaction'

export interface LogFilterOptions {
  readonly user?: string
  readonly command?: string
  readonly failed?: boolean
  readonly private?: boolean
  readonly guild?: string
  readonly platform?: string
  readonly authority?: number
  readonly since?: string
  readonly until?: string
}

export interface LogStatsOptions extends LogFilterOptions {
  readonly limit?: number
}

export function filterLogs(logs: CommandLogRecord[], options: LogFilterOptions): CommandLogRecord[] {
  let filteredLogs = logs

  if (options.user) {
    filteredLogs = filteredLogs.filter(log => log.userId === options.user)
  }
  if (options.command) {
    filteredLogs = filteredLogs.filter(log =>
      log.command.toLowerCase().includes(options.command.toLowerCase()))
  }
  if (options.failed) {
    filteredLogs = filteredLogs.filter(log => !log.success)
  }
  if (options.private) {
    filteredLogs = filteredLogs.filter(log => log.isPrivate)
  }
  if (options.guild) {
    filteredLogs = filteredLogs.filter(log => log.guildId === options.guild)
  }
  if (options.platform) {
    filteredLogs = filteredLogs.filter(log => log.platform === options.platform)
  }
  if (options.authority !== undefined) {
    filteredLogs = filteredLogs.filter(log => log.userAuthority === options.authority)
  }
  if (options.since) {
    filteredLogs = filterSince(filteredLogs, options.since)
  }
  if (options.until) {
    filteredLogs = filterUntil(filteredLogs, options.until)
  }

  return filteredLogs
}

export function formatLogList(logs: CommandLogRecord[], total: number): string {
  let message = `命令执行记录 (${logs.length}/${total})\n\n`
  logs.forEach((log, index) => {
    message += formatLogItem(log, index)
  })
  return message.trim()
}

export function formatStats(logs: CommandLogRecord[], options: LogStatsOptions): string {
  const commandGroups = groupLogsByCommand(logs)
  const sortedCommands = Array.from(commandGroups.entries())
    .sort((a, b) => b[1].length - a[1].length)
    .slice(0, options.limit)

  let message = `📊 命令使用统计\n`
  message += `总记录: ${logs.length} 条，命令种类: ${commandGroups.size} 个\n\n`

  sortedCommands.forEach(([command, cmdLogs], cmdIndex) => {
    const successCount = cmdLogs.filter(log => log.success).length
    const successRate = ((successCount / cmdLogs.length) * 100).toFixed(1)
    const lastUsed = getLastUsedLabel(cmdLogs)
    message += `${cmdIndex + 1}. ${command}\n`
    message += `   总计: ${cmdLogs.length} 次, 成功率: ${successRate}%\n`
    message += `   最后使用: ${lastUsed}\n\n`
  })

  return message.trim()
}

function filterSince(logs: CommandLogRecord[], since: string): CommandLogRecord[] {
  const sinceTime = new Date(since).getTime()
  if (isNaN(sinceTime)) return logs
  return logs.filter(log => new Date(log.timestamp).getTime() >= sinceTime)
}

function filterUntil(logs: CommandLogRecord[], until: string): CommandLogRecord[] {
  const untilTime = new Date(until).getTime()
  if (isNaN(untilTime)) return logs
  return logs.filter(log => new Date(log.timestamp).getTime() <= untilTime)
}

function formatLogItem(log: CommandLogRecord, index: number): string {
  const safeLog = redactCommandLogRecord(log)
  const status = safeLog.success ? '✅' : '❌'
  const time = new Date(safeLog.timestamp).toLocaleString('zh-CN')
  const location = safeLog.isPrivate ? '私聊' : `群(${safeLog.guildId})`
  const execTime = safeLog.executionTime > 0 ? ` (${safeLog.executionTime}ms)` : ''
  const authority = safeLog.userAuthority ? ` [权限:${safeLog.userAuthority}]` : ''
  let message = `${index + 1}. ${status} ${safeLog.command}${execTime}\n`

  message += `   用户: ${safeLog.username}(${safeLog.userId})${authority}\n`
  message += `   位置: ${location}\n`
  message += `   平台: ${safeLog.platform}\n`
  message += `   时间: ${time}\n`
  message += formatLogDetails(safeLog)
  return `${message}\n`
}

function formatLogDetails(log: CommandLogRecord): string {
  let message = ''
  if (log.args.length > 0) {
    message += `   参数: ${log.args.join(', ')}\n`
  }
  if (Object.keys(log.options).length > 0) {
    message += `   选项: ${JSON.stringify(log.options)}\n`
  }
  if (!log.success && log.error) {
    message += `   错误: ${log.error}\n`
  }
  return message
}

function groupLogsByCommand(logs: CommandLogRecord[]): Map<string, CommandLogRecord[]> {
  const commandGroups = new Map<string, CommandLogRecord[]>()
  logs.forEach(log => {
    let commandLogs = commandGroups.get(log.command)
    if (!commandLogs) {
      commandLogs = []
      commandGroups.set(log.command, commandLogs)
    }
    commandLogs.push(log)
  })
  return commandGroups
}

function getLastUsedLabel(logs: CommandLogRecord[]): string {
  return new Date(Math.max(...logs.map(log => new Date(log.timestamp).getTime())))
    .toLocaleString('zh-CN')
}
