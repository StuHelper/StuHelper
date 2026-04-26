import type { Command, Context } from 'koishi'

import type { DataManager } from '../data'
import type { Config } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { getRequiredPluginConfig } from './module-config'
import { registerOrderManageCommands } from './order-manage-commands'

export class OrderManageModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'manage-order',
    description: '秩序管理命令模块',
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
      registerOrderManageCommands(this)
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

  logCommand(session: any, command: string, target: string, result: string, success?: boolean): void {
    if (success === false) {
      session['_commandFailed'] = true
    }
    void this.ctx.stuhelperGroupCenter.logCommand(session, command, target, result)
  }

  recordMute(guildId: string, userId: string, duration: number): void {
    const mutes = this.data.mutes.getAll()
    if (!mutes[guildId]) {
      mutes[guildId] = {}
    }
    mutes[guildId][userId] = {
      startTime: Date.now(),
      duration,
    }
    this.data.mutes.setAll(mutes)
  }

  getRandomElements(arr: string[], n: number): string[] {
    const result: string[] = []
    const arrCopy = [...arr]
    n = Math.min(n, arrCopy.length)
    for (let i = 0; i < n; i++) {
      const idx = Math.floor(Math.random() * arrCopy.length)
      result.push(arrCopy[idx])
      arrCopy.splice(idx, 1)
    }
    return result
  }
}

export const orderManageRuntimeModule: RuntimeModule<OrderManageModule> = {
  id: 'manage-order',
  create(ctx, deps) {
    return new OrderManageModule(ctx, deps.data)
  },
}
