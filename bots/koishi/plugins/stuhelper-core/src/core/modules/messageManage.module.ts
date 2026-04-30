/**
 * messageManageModule - 基础群管命令模块
 *
 * 包含消息管理类群管功能：
 * - delmsg: 撤回消息
 */

import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'

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
    readonly data: DataManager
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
      registerDelMsgCommand(this)
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

async function handleDelMsgCommand(session: Session): Promise<string> {
  if (!session.quote) return '喵喵！请回复要撤回的消息呀~'
  const messageId = resolveQuoteMessageId(session.quote)
  if (!messageId) return '喵喵！无法读取被回复消息的 ID 呀~'

  try {
    await session.bot.deleteMessage(session.channelId, messageId)
    return ''
  } catch {
    return '呜呜...撤回失败了，可能太久了或者没有权限喵...'
  }
}

function resolveQuoteMessageId(quote: Session['quote']): string {
  if (!quote) return ''
  const value = quote.id || quote.messageId
  return value ? String(value) : ''
}

export const messageManageRuntimeModule: RuntimeModule<MessageManageModule> = {
  id: 'manage-message',
  create(ctx, deps) {
    return new MessageManageModule(ctx, deps.data)
  },
}
