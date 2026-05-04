/**
 * 欢迎模块 - 入群欢迎语管理
 */
import type { Context, Session } from 'koishi'

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
import { registerWelcomeCommands } from './welcome-commands'
import { registerWelcomeEventListeners } from './welcome-events'

export interface WelcomeLogEntry {
  session: Session
  command: string
  target: string
  result: string
}

export class WelcomeModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'welcome',
    description: '入群欢迎语管理模块',
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
        registerWelcomeCommands(this)
        registerWelcomeEventListeners(this)
      this.ctx.logger('stuhelper-core:welcome').info('WelcomeModule initialized')
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

  registerCommand(def: RuntimeCommandDef) {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }

  getGroupConfigs(): Record<string, GroupConfig> {
    return this.data.groupConfig.getAll()
  }

  setGroupConfigs(configs: Record<string, GroupConfig>): void {
    this.data.groupConfig.setAll(configs)
  }

  async log(entry: WelcomeLogEntry): Promise<void> {
    await this.ctx.stuhelperGroupCenter.logCommand(
      entry.session,
      entry.command,
      entry.target,
      entry.result,
    )
  }

  formatWelcomeMessage(template: string, userId: string, guildId: string): string {
    return template
      .replace(/{at}/g, `<at id="${userId}"/>`)
      .replace(/{user}/g, userId)
      .replace(/{group}/g, guildId)
  }
}

export const welcomeRuntimeModule: RuntimeModule<WelcomeModule> = {
  id: 'welcome',
  create(ctx, deps) {
    return new WelcomeModule(ctx, deps.data)
  },
}
