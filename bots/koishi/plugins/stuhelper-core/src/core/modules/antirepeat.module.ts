/**
 * BasicModule - 基础群管命令模块
 * - antirepeat: 复读管理
 */

import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { Config, GroupConfig } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { getRequiredPluginConfig } from './module-config'

const DEFAULT_ANTIREPEAT_THRESHOLD = 5
const MIN_ANTIREPEAT_THRESHOLD = 3

interface AntirepeatLogEntry {
  readonly session: Session
  readonly target: string
  readonly result: string
  readonly success?: boolean
}

export class AntirepeatModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'antirepeat',
    description: '防复读命令模块',
    version: '1.1',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager
  ) {}

  get config(): Config {
    return getRequiredPluginConfig(this.ctx)
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
      registerAntiRepeatCommand(this)
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

function registerAntiRepeatCommand(host: AntirepeatModule): void {
  registerRuntimeCommand(host.ctx, host.meta, {
    name: 'antirepeat',
    desc: '复读管理',
    args: '[threshold:number]',
    permNode: 'antirepeat',
    permDesc: '管理复读检测',
    usage: '设置复读阈值并启用，0为关闭',
    examples: ['antirepeat 5', 'antirepeat 0'],
  })
    .action(async ({ session }, threshold) => {
      return handleAntirepeatCommand(host, session, threshold)
    })
}

async function handleAntirepeatCommand(
  host: AntirepeatModule,
  session: Session,
  threshold?: number,
): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

  const groupConfigs = host.data.groupConfig.getAll()
  const groupConfig = groupConfigs[session.guildId] || {}
  const antiRepeatConfig = groupConfig.antiRepeat || {
    enabled: false,
    threshold: host.config.antiRepeat?.threshold || DEFAULT_ANTIREPEAT_THRESHOLD,
  }

  if (threshold === undefined) {
    return formatCurrentConfig(antiRepeatConfig)
  }

  if (threshold === 0) {
    return disableAntirepeat({
      host,
      session,
      groupConfigs,
      groupConfig,
      threshold: antiRepeatConfig.threshold,
    })
  }

  if (threshold < MIN_ANTIREPEAT_THRESHOLD) {
    logAntirepeatAction(host, {
      session,
      target: session.guildId,
      result: '失败：无效的阈值',
      success: false,
    })
    return '喵呜...阈值至少要设置为3条以上喵...'
  }

  enableAntirepeat({ host, session, groupConfigs, groupConfig, threshold })
  return `已设置本群复读阈值为 ${threshold} 条并启用检测喵~`
}

interface AntirepeatConfigUpdateInput {
  readonly host: AntirepeatModule
  readonly session: Session
  readonly groupConfigs: Record<string, GroupConfig>
  readonly groupConfig: GroupConfig
  readonly threshold: number
}

function disableAntirepeat(input: AntirepeatConfigUpdateInput) {
  const { host, session, groupConfigs, groupConfig, threshold } = input
  groupConfigs[session.guildId!] = {
    ...groupConfig,
    antiRepeat: { enabled: false, threshold },
  }
  host.data.groupConfig.setAll(groupConfigs)
  logAntirepeatAction(host, { session, target: session.guildId!, result: '成功：已关闭复读检测' })
  return '已关闭本群的复读检测喵~'
}

function enableAntirepeat(input: AntirepeatConfigUpdateInput) {
  const { host, session, groupConfigs, groupConfig, threshold } = input
  groupConfigs[session.guildId!] = {
    ...groupConfig,
    antiRepeat: { enabled: true, threshold },
  }
  host.data.groupConfig.setAll(groupConfigs)
  logAntirepeatAction(host, {
    session,
    target: session.guildId!,
    result: `成功：已设置阈值为 ${threshold} 并启用`,
  })
}

function formatCurrentConfig(config: NonNullable<GroupConfig['antiRepeat']>): string {
  return `当前群复读配置：
状态：${config.enabled ? '已启用' : '未启用'}
阈值：${config.threshold} 条
使用方法：
antirepeat 数字 - 设置复读阈值并启用（至少3条）
antirepeat 0 - 关闭复读检测`
}

function logAntirepeatAction(
  host: AntirepeatModule,
  entry: AntirepeatLogEntry,
): void {
  if (entry.success === false) {
    entry.session['_commandFailed'] = true
  }
  host.ctx.stuhelperGroupCenter.logCommand({
    session: entry.session,
    command: 'antirepeat',
    target: entry.target,
    result: entry.result,
  })
}

export const antirepeatRuntimeModule: RuntimeModule<AntirepeatModule> = {
  id: 'antirepeat',
  create(ctx, deps) {
    return new AntirepeatModule(ctx, deps.data)
  },
}
