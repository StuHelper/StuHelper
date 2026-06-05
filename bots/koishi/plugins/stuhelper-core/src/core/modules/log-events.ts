import type { Session } from 'koishi'

import {
  clearCommandExecutionState,
  commandExecutionDuration,
  getCommandExecutionState,
  markCommandExecutionStarted,
} from './command-execution-state'
import type { CommandLogRecord, LogModule } from './log.module'

const DEFAULT_USER_AUTHORITY = 1
const RANDOM_ID_RADIX = 36
const RANDOM_ID_START = 2
const RANDOM_ID_LENGTH = 9

interface CommandExecutionInput {
  argv: CommandArgvLike
  success: boolean
  error?: string
  result?: unknown
}

interface CommandLogRecordInput {
  argv: CommandArgvLike
  session: Session
  success: boolean
  error?: string
  result?: unknown
  userAuthority: number
}

interface CommandArgvLike {
  readonly session?: Session
  readonly command?: {
    readonly name?: string
  }
  readonly name?: string
  readonly args?: readonly unknown[]
  readonly options?: unknown
}

export function registerLogEventListeners(host: LogModule): void {
  host.ctx.on('command/before-execute', (argv) => {
    markCommandExecutionStarted(argv)
  })

  host.ctx.on('command-error', (argv, error) => {
    const commandArgv = toCommandArgv(argv)
    if (!commandArgv) return

    void recordCommandExecution(host, {
      argv: commandArgv,
      success: false,
      error: error?.message || 'Unknown error',
    })
  })

  host.ctx.middleware(async (session, next) => {
    const result = await next()
    const commandArgv = toCommandArgv(session.argv)
    if (commandArgv?.command) {
      const commandState = getCommandExecutionState(session)
      void recordCommandExecution(host, {
        argv: commandArgv,
        success: !commandState.failed,
        error: commandState.error,
        result,
      }).finally(() => clearCommandExecutionState(session))
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
  const executionTime = commandExecutionDuration(input.argv)
  const session = input.session

  return {
    id: `${Date.now()}_${Math.random().toString(RANDOM_ID_RADIX).substr(RANDOM_ID_START, RANDOM_ID_LENGTH)}`,
    timestamp: new Date().toISOString(),
    userId: session.userId || 'unknown',
    username: session.username || session.author?.nickname || session.author?.username || 'unknown',
    userAuthority: input.userAuthority,
    guildId: session.guildId,
    guildName: readGuildName(session),
    channelId: session.channelId,
    platform: session.platform || 'unknown',
    command: input.argv.command?.name || input.argv.name || 'unknown',
    args: readArgs(input.argv),
    options: readOptions(input.argv),
    success: input.success,
    error: input.error,
    executionTime,
    result: typeof input.result === 'string' ? input.result : undefined,
    messageId: session.messageId,
    isPrivate: !session.guildId,
  }
}

function toCommandArgv(value: unknown): CommandArgvLike | null {
  if (!isRecord(value)) return null
  return value
}

function readArgs(argv: CommandArgvLike): string[] {
  if (!Array.isArray(argv.args)) return []
  return argv.args.map((value) => String(value))
}

function readOptions(argv: CommandArgvLike): Record<string, unknown> {
  return isRecord(argv.options) ? argv.options : {}
}

function readGuildName(session: Session): string | undefined {
  const guild: unknown = session.guild
  if (!isRecord(guild)) return undefined

  return typeof guild.name === 'string' ? guild.name : undefined
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
