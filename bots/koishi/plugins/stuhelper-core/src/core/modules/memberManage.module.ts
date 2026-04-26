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

  logCommand(session: any, command: string, target: string, result: string, success?: boolean): void {
    if (success === false) {
      session['_commandFailed'] = true
    }
    void this.ctx.stuhelperGroupCenter.logCommand(session, command, target, result)
  }
}

export const memberManageRuntimeModule: RuntimeModule<MemberManageModule> = {
  id: 'manage-member',
  create(ctx, deps) {
    return new MemberManageModule(ctx, deps.data, deps.config)
  },
}
