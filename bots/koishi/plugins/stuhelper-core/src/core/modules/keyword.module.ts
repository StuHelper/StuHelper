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
import { registerKeywordForbiddenCommand } from './keyword-forbidden-command'
import { registerKeywordMiddleware } from './keyword-middleware'
import { registerKeywordVerifyCommand } from './keyword-verify-command'

export class KeywordModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'keyword',
    description: '关键词管理功能，包括入群验证和禁言关键词',
    version: '1.0.1',
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
      registerKeywordVerifyCommand(this)
      registerKeywordForbiddenCommand(this)
      registerKeywordMiddleware(this)
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

  getVerifyKeywords(guildId: string): string[] {
    const groupConfig = this.data.groupConfig.get(guildId) || {} as GroupConfig
    return groupConfig.approvalKeywords || []
  }

  getForbiddenKeywords(guildId: string): string[] {
    const groupConfig = this.data.groupConfig.get(guildId) || {} as GroupConfig
    return groupConfig.keywords || []
  }

  getEffectiveKeywords(guildId: string): string[] {
    const groupKeywords = this.getForbiddenKeywords(guildId)
    return [...this.config.forbidden.keywords, ...groupKeywords]
  }

  matchKeyword(content: string, keyword: string): boolean {
    try {
      const regex = new RegExp(keyword, 'i')
      return regex.test(content)
    } catch {
      return content.includes(keyword)
    }
  }
}

export const keywordRuntimeModule: RuntimeModule<KeywordModule> = {
  id: 'keyword',
  create(ctx, deps) {
    return new KeywordModule(ctx, deps.data, deps.config)
  },
}
