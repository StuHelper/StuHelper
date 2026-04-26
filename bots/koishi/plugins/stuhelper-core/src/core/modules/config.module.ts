/**
 * 配置管理模块
 * 提供配置查看、黑名单管理、警告管理功能
 */

import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { Config } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommand,
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { registerConfigCommands } from './config-commands'

export class ConfigModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'config',
    description: '配置管理模块',
    version: '1.0.0',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager,
    private readonly initialConfig: Config,
  ) {}

  get config(): Config {
    try {
      return this.ctx.stuhelperGroupCenter?.pluginConfig || this.initialConfig
    } catch {
      return this.initialConfig
    }
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
      registerConfigCommands(this)
      console.log(`[${this.meta.name}] ConfigModule initialized`)
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

  async log(
    session: Session,
    command: string,
    target: string,
    result: string,
    success?: boolean,
  ): Promise<void> {
    if (success === false) {
      session['_commandFailed'] = true
    }
    await this.ctx.stuhelperGroupCenter.logCommand(session, command, target, result)
  }
}

export const configRuntimeModule: RuntimeModule<ConfigModule> = {
  id: 'config',
  create(ctx, deps) {
    return new ConfigModule(ctx, deps.data, deps.config)
  },
}
