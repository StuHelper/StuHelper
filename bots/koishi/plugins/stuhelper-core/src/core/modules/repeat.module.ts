import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { sleep } from '../../utils'

const REPEAT_DELETE_DELAY_MS = 300
const REPEAT_RECORD_TTL_MS = 60 * 60 * 1000
const DEFAULT_REPEAT_THRESHOLD = 3

/**
 * 复读记录接口
 */
interface RepeatRecord {
  content: string
  count: number
  firstMessageId: string
  messages: Array<{
    id: string
    userId: string
    timestamp: number
  }>
}

/**
 * 复读机检测模块
 * 检测并处理群内复读消息
 */
export class RepeatModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'repeat',
    description: '复读机检测模块',
    version: '1.0.0',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null
  private readonly repeatMap = new Map<string, RepeatRecord>()
  private cleanupTimer: NodeJS.Timeout | null = null

  constructor(
    readonly ctx: Context,
    private readonly data: DataManager,
  ) {}

  get state(): RuntimeModuleState {
    return this._state
  }

  get error(): Error | null {
    return this._error
  }

  async init(): Promise<void> {
    this._state = 'loading'
    try {
      registerMiddleware(this)
      setupCleanupTask(this)
      this.ctx.logger.info('[RepeatModule] initialized')
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    this._state = 'unloaded'
  }

  getRepeatMap(): Map<string, RepeatRecord> {
    return this.repeatMap
  }

  getRepeatRecord(guildId: string): RepeatRecord | undefined {
    return this.repeatMap.get(guildId)
  }

  clearRepeatRecord(guildId: string): void {
    this.repeatMap.delete(guildId)
  }

  getGroupConfig(guildId: string) {
    return this.data.groupConfig.get(guildId)
  }

  clearCleanupTimer(): void {
    if (!this.cleanupTimer) return
    clearInterval(this.cleanupTimer)
    this.cleanupTimer = null
  }

  setCleanupTimer(timer: NodeJS.Timeout): void {
    this.cleanupTimer = timer
  }
}

function registerMiddleware(host: RepeatModule): void {
  host.ctx.middleware(async (session, next) => {
    if (!session.content || !session.guildId) return next()

    const groupConfig = host.getGroupConfig(session.guildId)
    const antiRepeatConfig = groupConfig?.antiRepeat
    if (!antiRepeatConfig?.enabled) return next()

    await handleRepeatSession(host, session, antiRepeatConfig.threshold)
    return next()
  })
}

async function handleRepeatSession(
  host: RepeatModule,
  session: Session,
  threshold = DEFAULT_REPEAT_THRESHOLD,
): Promise<void> {
  if (!session.guildId || !session.content) return

  const repeatMap = host.getRepeatMap()
  const record = repeatMap.get(session.guildId)
  if (!record || record.content !== session.content) {
    repeatMap.set(session.guildId, createRepeatRecord(session, session.content))
    return
  }

  record.count++
  record.messages.push(createRepeatMessage(session))
  if (record.count <= threshold) return

  try {
    logRepeatDeletion(host, session, record.count - 1)
    await deleteRepeatedMessages(host, session, record)
    repeatMap.delete(session.guildId)
  } catch (error) {
    host.ctx.logger.error('[RepeatModule] 处理复读消息时出错:', error)
  }
}

function createRepeatRecord(session: Session, content: string): RepeatRecord {
  return {
    content,
    count: 1,
    firstMessageId: session.messageId,
    messages: [createRepeatMessage(session)],
  }
}

function createRepeatMessage(session: Session) {
  return {
    id: session.messageId,
    userId: session.userId,
    timestamp: Date.now(),
  }
}

function logRepeatDeletion(
  host: RepeatModule,
  session: Session,
  count: number,
): void {
  void host.ctx.stuhelperGroupCenter.logCommand({
    session,
    command: 'antirepeat',
    target: 'messages',
    result: `已删除 ${count} 条复读消息`,
  })
}

async function deleteRepeatedMessages(
  host: RepeatModule,
  session: Session,
  record: RepeatRecord,
): Promise<void> {
  for (let index = 1; index < record.messages.length; index++) {
    const message = record.messages[index]
    try {
      await session.bot.deleteMessage(session.channelId, message.id)
    } catch (error) {
      host.ctx.logger.error(`[RepeatModule] 撤回消息 ${message.id} 失败:`, error)
    }
    await sleep(REPEAT_DELETE_DELAY_MS)
  }
}

function setupCleanupTask(host: RepeatModule): void {
  host.setCleanupTimer(setInterval(() => {
    pruneExpiredRecords(host.getRepeatMap())
  }, REPEAT_RECORD_TTL_MS))

  host.ctx.on('dispose', () => {
    host.clearCleanupTimer()
  })
}

function pruneExpiredRecords(repeatMap: Map<string, RepeatRecord>): void {
  const now = Date.now()
  for (const [guildId, record] of repeatMap.entries()) {
    const latestMessage = record.messages[record.messages.length - 1]
    if (now - latestMessage.timestamp > REPEAT_RECORD_TTL_MS) {
      repeatMap.delete(guildId)
    }
  }
}

export const repeatRuntimeModule: RuntimeModule<RepeatModule> = {
  id: 'repeat',
  create(ctx, deps) {
    return new RepeatModule(ctx, deps.data)
  },
}
