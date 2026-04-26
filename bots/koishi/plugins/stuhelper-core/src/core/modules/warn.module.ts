import type { Command, Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { Config, GroupConfig } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { getRequiredPluginConfig } from './module-config'
import { registerWarnCommands } from './warn-commands'
import { migrateWarnData } from './warn-migration'

export interface WarnLogInput {
  readonly session: Session
  readonly command: string
  readonly target: string
  readonly result: string
}

export class WarnModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'warn',
    description: '用户警告功能，累计警告达到阈值时自动禁言',
    version: '1.0.0',
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
      migrateWarnData(this)
      registerWarnCommands(this)
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

  registerCommand(def: RuntimeCommandDef): Command {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }

  async log(input: WarnLogInput): Promise<void> {
    await this.ctx.stuhelperGroupCenter.logCommand(
      input.session,
      input.command,
      input.target,
      input.result,
    )
  }

  getGroupConfig(guildId: string): GroupConfig | undefined {
    return this.data.groupConfig.get(guildId)
  }

  addWarn(guildId: string, userId: string, count: number): number {
    const guildWarns = this.data.warns.get(guildId) || {}
    if (!guildWarns[userId]) {
      guildWarns[userId] = { count: 0, timestamp: Date.now() }
    }

    guildWarns[userId].count += count
    guildWarns[userId].timestamp = Date.now()
    this.data.warns.set(guildId, guildWarns)
    this.data.warns.flush()
    return guildWarns[userId].count
  }

  recordMute(guildId: string, userId: string, duration: number): void {
    const guildMutes = this.data.mutes.get(guildId) || {}
    guildMutes[userId] = {
      startTime: Date.now(),
      duration,
      remainingTime: duration,
    }
    this.data.mutes.set(guildId, guildMutes)
    this.data.mutes.flush()
  }

  getWarnCount(guildId: string, userId: string): number {
    const guildWarns = this.data.warns.get(guildId)
    return guildWarns?.[userId]?.count || 0
  }

  getGuildWarns(guildId: string): Array<{ userId: string; count: number; timestamp: number }> {
    const result: Array<{ userId: string; count: number; timestamp: number }> = []
    const guildWarns = this.data.warns.get(guildId)
    if (!guildWarns) return result

    for (const [userId, record] of Object.entries(guildWarns)) {
      result.push({
        userId,
        count: record.count,
        timestamp: record.timestamp,
      })
    }
    return result
  }
}

export const warnRuntimeModule: RuntimeModule<WarnModule> = {
  id: 'warn',
  create(ctx, deps) {
    return new WarnModule(ctx, deps.data)
  },
}
