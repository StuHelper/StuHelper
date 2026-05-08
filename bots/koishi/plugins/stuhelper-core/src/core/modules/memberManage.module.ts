import type { Command, Context } from 'koishi'
import { createPlatformClient, type PlatformClient } from '@stuhelper/koishi-shared'

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
import { registerMemberManageCommands } from './member-manage-commands'

export class MemberManageModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'manage-member',
    description: '成员管理模块',
    version: '1.1',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager,
    readonly memberBlacklistBackend: Pick<PlatformClient, 'createMemberBlacklist'>,
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
      registerMemberManageCommands(this)
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

  logCommand(entry: {
    readonly session: any
    readonly command: string
    readonly target: string
    readonly result: string
    readonly success?: boolean
  }): void {
    const { session, command, target, result, success } = entry
    if (success === false) {
      session['_commandFailed'] = true
    }
    void this.ctx.stuhelperGroupCenter.logCommand({ session, command, target, result })
  }
}

export const memberManageRuntimeModule: RuntimeModule<MemberManageModule> = {
  id: 'manage-member',
  create(ctx, deps) {
    if (!deps.coreConfig) throw new Error('stuhelper core config is required for MemberManageModule')
    return new MemberManageModule(ctx, deps.data, createPlatformClient(deps.coreConfig.platform))
  },
}
