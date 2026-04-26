/**
 * messageManageModule - 基础群管命令模块
 *
 * 包含消息管理类群管功能：
 * - delmsg: 撤回消息
 * - essence: 精华消息
 */

import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { Config } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'

const DEFAULT_ESSENCE_AUTHORITY = 3
const DEFAULT_ESSENCE_CONFIG: Config['setEssenceMsg'] = {
  enabled: false,
  authority: DEFAULT_ESSENCE_AUTHORITY,
}

interface EssenceCommandInput {
  readonly host: MessageManageModule
  readonly session: Session
  readonly options: {
    readonly s?: boolean
    readonly r?: boolean
  }
  readonly config: Config['setEssenceMsg']
}

interface CommandLogEntry {
  readonly session: Session
  readonly command: string
  readonly target: string
  readonly result: string
  readonly success?: boolean
}

export class MessageManageModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'manage-message',
    description: '消息管理模块',
    version: '1.1',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager,
    private readonly initialConfig: Config,
  ) {}

  get config(): Config {
    return this.ctx.stuhelperGroupCenter?.pluginConfig || this.initialConfig
  }

  get state(): RuntimeModuleState {
    return this._state
  }

  get error(): Error | null {
    return this._error
  }

  async init(): Promise<void> {
    this._state = 'loading'
    try {
      registerDelMsgCommand(this)
      registerEssenceCommand(this)
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
}

function registerDelMsgCommand(host: MessageManageModule): void {
  registerRuntimeCommand(host.ctx, host.meta, {
    name: 'delmsg',
    desc: '撤回消息',
    permNode: 'delmsg',
    permDesc: '撤回群消息',
    usage: '回复要撤回的消息后使用此命令',
  })
    .action(async ({ session }) => handleDelMsgCommand(session))
}

function registerEssenceCommand(host: MessageManageModule): void {
  const essenceConfig = host.config.setEssenceMsg || DEFAULT_ESSENCE_CONFIG

  registerRuntimeCommand(host.ctx, host.meta, {
    name: 'essence',
    desc: '精华消息管理',
    permNode: 'essence',
    permDesc: '管理精华消息',
    usage: '-s 设置精华消息，-r 取消精华消息',
    examples: ['essence -s (回复消息)', 'essence -r (回复消息)'],
  })
    .option('s', '-s 设置精华消息')
    .option('r', '-r 取消精华消息')
    .action(async ({ session, options }) => {
      return handleEssenceCommand({
        host,
        session,
        options,
        config: essenceConfig,
      })
    })
}

async function handleDelMsgCommand(session: Session): Promise<string> {
  if (!session.quote) return '喵喵！请回复要撤回的消息呀~'

  try {
    await session.bot.deleteMessage(session.channelId, session.quote.id)
    return ''
  } catch {
    return '呜呜...撤回失败了，可能太久了或者没有权限喵...'
  }
}

async function handleEssenceCommand(input: EssenceCommandInput): Promise<string> {
  const { host, session, options, config } = input
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
  if (!config.enabled) return '喵呜...精华消息功能未启用...'
  if (!session.quote) return '喵喵！请回复要操作的消息呀~'

  try {
    if (options.s) return await setEssenceMessage(host, session)
    if (options.r) return await removeEssenceMessage(host, session)
    return '请使用 -s 设置精华消息或 -r 取消精华消息'
  } catch (error) {
    logCommand(host, {
      session,
      command: 'essence',
      target: session.quote?.messageId || 'none',
      result: '失败：未知错误',
      success: false,
    })
    return `出错啦喵...${getErrorMessage(error)}`
  }
}

async function setEssenceMessage(
  host: MessageManageModule,
  session: Session,
): Promise<string> {
  await (session.bot as any).internal.setEssenceMsg(session.quote.messageId)
  logCommand(host, {
    session,
    command: 'essence',
    target: 'set',
    result: `成功：已设置精华消息：${session.quote.messageId}`,
  })
  return '已经设置为精华消息啦喵~'
}

async function removeEssenceMessage(
  host: MessageManageModule,
  session: Session,
): Promise<string> {
  await (session.bot as any).internal.deleteEssenceMsg(session.quote.messageId)
  logCommand(host, {
    session,
    command: 'essence',
    target: 'remove',
    result: `成功：已取消精华消息：${session.quote.messageId}`,
  })
  return '已经取消精华消息啦喵~'
}

function logCommand(
  host: MessageManageModule,
  entry: CommandLogEntry,
): void {
  if (entry.success === false) {
    entry.session['_commandFailed'] = true
  }
  host.ctx.stuhelperGroupCenter.logCommand(
    entry.session,
    entry.command,
    entry.target,
    entry.result,
  )
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error)
}

export const messageManageRuntimeModule: RuntimeModule<MessageManageModule> = {
  id: 'manage-message',
  create(ctx, deps) {
    return new MessageManageModule(ctx, deps.data, deps.config)
  },
}
