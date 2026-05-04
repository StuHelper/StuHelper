import type { Session } from 'koishi'

import type { CommandLogRecord, LogModule } from './log.module'

const DEFAULT_USER_AUTHORITY = 1
const RANDOM_ID_RADIX = 36
const RANDOM_ID_START = 2
const RANDOM_ID_LENGTH = 9

interface CommandExecutionInput {
  argv: any
  success: boolean
  error?: string
  result?: any
}

interface CommandLogRecordInput {
  argv: any
  session: Session
  success: boolean
  error?: string
  result?: any
  userAuthority: number
}

export function registerLogEventListeners(host: LogModule): void {
  host.ctx.on('command/before-execute', (argv) => {
    ;(argv as any)._startTime = Date.now()
  })

  host.ctx.on('command-error', (argv, error) => {
    void recordCommandExecution(host, {
      argv,
      success: false,
      error: error?.message || 'Unknown error',
    })
  })

  host.ctx.middleware(async (session, next) => {
    const result = await next()
    if (session.argv?.command) {
      const commandFailed = (session as any)._commandFailed
      const commandError = (session as any)._commandError
      void recordCommandExecution(host, {
        argv: session.argv,
        success: !commandFailed,
        error: commandFailed ? commandError : undefined,
        result,
      })
    }
    return result
  }, true)
}

async function recordCommandExecution(
  host: LogModule,
  input: CommandExecutionInput,
): Promise<void> {
  try {
    const session = input.argv.session
    if (!session) return

    const userAuthority = await resolveUserAuthority(host, session)
    const logRecord = buildCommandLogRecord({ ...input, session, userAuthority })
    const logs = host.readCommandLogs()
    logs.push(logRecord)
    host.saveCommandLogs(logs)
    host.recordCommandUsage(logRecord.command)
  } catch (error) {
    host.ctx.logger('stuhelper-core:log').error('记录命令日志失败: %o', error)
  }
}

async function resolveUserAuthority(host: LogModule, session: Session): Promise<number> {
  if (!host.ctx.database) return DEFAULT_USER_AUTHORITY

  try {
    const user = await session.observeUser(['authority'])
    return user?.authority || DEFAULT_USER_AUTHORITY
  } catch {
    return DEFAULT_USER_AUTHORITY
  }
}

function buildCommandLogRecord(input: CommandLogRecordInput): CommandLogRecord {
  const executionTime = input.argv._startTime ? Date.now() - input.argv._startTime : 0
  const session = input.session

  return {
    id: `${Date.now()}_${Math.random().toString(RANDOM_ID_RADIX).substr(RANDOM_ID_START, RANDOM_ID_LENGTH)}`,
    timestamp: new Date().toISOString(),
    userId: session.userId || 'unknown',
    username: session.username || session.author?.nickname || session.author?.username || 'unknown',
    userAuthority: input.userAuthority,
    guildId: session.guildId,
    guildName: (session.guild as any)?.name,
    channelId: session.channelId,
    platform: session.platform || 'unknown',
    command: input.argv.command?.name || input.argv.name || 'unknown',
    args: input.argv.args || [],
    options: input.argv.options || {},
    success: input.success,
    error: input.error,
    executionTime,
    result: typeof input.result === 'string' ? input.result : undefined,
    messageId: session.messageId,
    isPrivate: !session.guildId,
  }
}
