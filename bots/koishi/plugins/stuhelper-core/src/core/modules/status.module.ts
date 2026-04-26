/**
 * StatusModule - 状态模块
 * 提供系统状态查询和可视化展示
 */

import type { Context } from 'koishi'

import type { DataManager } from '../data'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommand,
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { registerStatusCommands } from './status-commands'

export class StatusModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'status',
    description: '状态模块 - 查看 Bot 运行状态',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager,
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
      registerStatusCommands(this)
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

  registerCommand(def: RuntimeCommandDef): RuntimeCommand {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }
}

export const statusRuntimeModule: RuntimeModule<StatusModule> = {
  id: 'status',
  create(ctx, deps) {
    return new StatusModule(ctx, deps.data)
  },
}
