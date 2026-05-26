import type { Session } from 'koishi'
import * as fs from 'fs'

import type { LogModule } from './log.module'

const DEFAULT_OPERATION_LOG_LINES = 100
const DEFAULT_KEEP_DAYS = 7
const DAY_MS = 24 * 60 * 60 * 1000

interface ClearOperationLogOptions {
  readonly a?: boolean
  readonly d?: number
}

export function registerOperationLogCommands(host: LogModule): void {
  registerListLogCommand(host)
  registerClearLogCommand(host)
}

function registerListLogCommand(host: LogModule): void {
  host.registerCommand({
    name: 'listlog',
    desc: '显示最近的操作记录',
    args: '[lines:number]',
    permNode: 'listlog',
    permDesc: '查看操作日志',
    usage: '显示最近的操作日志，可指定条数',
    examples: ['listlog', 'listlog 50'],
  })
    .action(async ({ session }, lines = DEFAULT_OPERATION_LOG_LINES) => {
      return handleListLog(host, session, lines)
    })
}

function registerClearLogCommand(host: LogModule): void {
  host.registerCommand({
    name: 'clearlog',
    desc: '清理日志文件',
    permNode: 'clearlog',
    permDesc: '清理操作日志',
    usage: '-d 天数 保留最近几天，-a 清理所有',
    examples: ['clearlog -d 7', 'clearlog -a'],
  })
    .option('d', '-d <days:number> 保留最近几天的日志')
    .option('a', '-a 清理所有日志')
    .action(async ({ session, options }) => handleClearLog(host, session, options))
}

function handleListLog(host: LogModule, session: Session, lines: number): string {
  if (!fs.existsSync(host.logPath)) return '还没有任何日志记录喵~'

  try {
    const content = fs.readFileSync(host.logPath, 'utf8')
    const allLines = content.split('\n').filter(line => line.trim())
    const recentLines = allLines.slice(-lines)
    if (recentLines.length === 0) return '还没有任何日志记录喵~'

    void host.logCommand({ session, command: 'listlog', target: `${lines}`, result: 'success' })
    return `=== 最近 ${Math.min(lines, recentLines.length)} 条操作记录 ===\n${recentLines.join('\n')}`
  } catch (error) {
    return `读取日志失败喵...${errorMessage(error)}`
  }
}

function handleClearLog(host: LogModule, session: Session, options: ClearOperationLogOptions): string {
  if (!fs.existsSync(host.logPath)) return '还没有任何日志记录喵~'

  try {
    if (options.a) {
      fs.writeFileSync(host.logPath, '')
      void host.logCommand({ session, command: 'clearlog', target: 'all', result: 'Cleared all logs' })
      return '已清理所有日志记录喵~'
    }

    const days = options.d || DEFAULT_KEEP_DAYS
    const deletedCount = keepRecentOperationLogs(host.logPath, days)
    void host.logCommand({
      session,
      command: 'clearlog',
      target: `${days}days`,
      result: `Cleared ${deletedCount} logs`,
    })
    return `已清理 ${deletedCount} 条日志记录，保留最近 ${days} 天的记录喵~`
  } catch (error) {
    return `清理日志失败喵...${errorMessage(error)}`
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function keepRecentOperationLogs(logPath: string, days: number): number {
  const now = Date.now()
  const content = fs.readFileSync(logPath, 'utf8')
  const allLines = content.split('\n').filter(line => line.trim())
  const keptLogs = allLines.filter(line => shouldKeepOperationLog(line, now, days))

  fs.writeFileSync(logPath, keptLogs.join('\n') + '\n')
  return allLines.length - keptLogs.length
}

function shouldKeepOperationLog(line: string, now: number, days: number): boolean {
  const match = line.match(/^\[(.*?)\]/)
  if (!match) return false
  const logTime = new Date(match[1]).getTime()
  return (now - logTime) <= days * DAY_MS
}
