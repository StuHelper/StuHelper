/**
 * 配置管理模块
 * 提供配置查看、黑名单管理、警告管理功能
 */

import type { Context, Session } from 'koishi'
import type { StuhelperPlatformConfig } from '@stuhelper/koishi-shared'

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
import { getRequiredPluginConfig } from './module-config'
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
    readonly platformConfig: StuhelperPlatformConfig,
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
        registerConfigCommands(this)
      this.ctx.logger('stuhelper-core:config').info('ConfigModule initialized')
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

  async log(entry: {
    readonly session: Session
    readonly command: string
    readonly target: string
    readonly result: string
    readonly success?: boolean
  }): Promise<void> {
    const { session, command, target, result, success } = entry
    if (success === false) {
      session['_commandFailed'] = true
    }
    await this.ctx.stuhelperGroupCenter.logCommand({ session, command, target, result })
  }
}

export const configRuntimeModule: RuntimeModule<ConfigModule> = {
  id: 'config',
  create(ctx, deps) {
    if (!deps.coreConfig) throw new Error('stuhelper core config is required for ConfigModule')
    return new ConfigModule(ctx, deps.data, deps.coreConfig.platform)
  },
}
