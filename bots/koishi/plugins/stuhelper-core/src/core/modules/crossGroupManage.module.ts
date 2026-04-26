/**
 * crossGroupModule - 跨群管理命令模块
 *
 * 包含核心群管功能：
 * - quit-group: 退出群聊
 * - send: 远程发送消息
 */

import type { Context, Session } from 'koishi'

import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'

interface CommandLogEntry {
  readonly session: Session
  readonly command: string
  readonly target: string
  readonly result: string
  readonly success?: boolean
}

export class crossGroupModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'manage-cross-group',
    description: '跨群管理命令模块',
    version: '1.1',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(private readonly ctx: Context) {}

  get state(): RuntimeModuleState {
    return this._state
  }

  get error(): Error | null {
    return this._error
  }

  async init(): Promise<void> {
    this._state = 'loading'
    try {
      registerQuitGroupCommand(this.ctx, this.meta)
      registerSendCommand(this.ctx, this.meta)
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

function registerQuitGroupCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'quit-group',
    desc: '退出指定群聊',
    args: '<groupId:string>',
    permNode: 'quit-group',
    permDesc: '退出群聊（高危）',
    usage: '让机器人退出指定的群聊',
    examples: ['quit-group 123456789'],
  })
    .example('quit-group 123456789')
    .action(async ({ session }, groupId) => {
      return handleQuitGroupCommand(ctx, session, groupId)
    })
}

function registerSendCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'send',
    desc: '向指定群发送消息',
    args: '<groupId:string>',
    permNode: 'send',
    permDesc: '远程发送群消息',
    usage: '回复一条消息后使用，-s 静默发送（不显示发送者）',
    examples: ['send 123456789', 'send 123456789 -s'],
  })
    .example('send 123456789')
    .option('s', '-s 静默发送，不显示发送者信息')
    .action(async ({ session, options }, groupId) => {
      return handleSendCommand(ctx, session, groupId, Boolean(options.s))
    })
}

async function handleQuitGroupCommand(
  ctx: Context,
  session: Session,
  groupId?: string,
): Promise<string> {
  if (!groupId) return '喵呜...请指定要退出的群聊ID喵~'

  try {
    await (session.bot as any).internal.setGroupLeave(groupId, false)
    logCommand(ctx, {
      session,
      command: 'quit-group',
      target: groupId,
      result: `成功：已退出群聊 ${groupId}`,
    })
    return `已成功退出群聊 ${groupId} 喵~`
  } catch (error) {
    logCommand(ctx, {
      session,
      command: 'quit-group',
      target: groupId,
      result: '失败：未知错误',
      success: false,
    })
    return `喵呜...退出群聊失败了：${getErrorMessage(error)}`
  }
}

async function handleSendCommand(
  ctx: Context,
  session: Session,
  groupId: string,
  silent: boolean,
): Promise<string> {
  if (!session.quote) return '喵喵！请回复要发送的消息呀~'

  try {
    const content = silent
      ? session.quote.content
      : `用户${session.userId}远程投送消息：\n${session.quote.content}`
    await session.bot.sendMessage(groupId, content)
    logCommand(ctx, buildSendLogEntry(session, groupId, silent))
    return `已将消息发送到群 ${groupId} 喵~`
  } catch (error) {
    logCommand(ctx, {
      session,
      command: 'send',
      target: groupId,
      result: '失败：未知错误',
      success: false,
    })
    return `喵呜...发送失败了：${getErrorMessage(error)}`
  }
}

function buildSendLogEntry(
  session: Session,
  groupId: string,
  silent: boolean,
): CommandLogEntry {
  return {
    session,
    command: 'send',
    target: groupId,
    result: silent
      ? `成功：已静默发送消息：${session.quote.messageId}`
      : `成功：已发送消息：${session.quote.messageId}`,
  }
}

function logCommand(ctx: Context, entry: CommandLogEntry): void {
  if (entry.success === false) {
    entry.session['_commandFailed'] = true
  }
  ctx.stuhelperGroupCenter.logCommand(
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

export const crossGroupRuntimeModule: RuntimeModule<crossGroupModule> = {
  id: 'manage-cross-group',
  create(ctx) {
    return new crossGroupModule(ctx)
  },
}
