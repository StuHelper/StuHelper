import type { Command, Context, Session } from 'koishi'
import * as path from 'path'

import type { DataManager } from '../data'
import type { ChatMessage, ChatCompletionResponse, Config, GroupConfig, UserContext } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import {
  addMessageToContext,
  cleanExpiredContexts,
  initDataFiles,
  loadContexts,
  saveContexts,
} from './ai-context'
import { callOpenAI, type OpenAIRequestInput } from './ai-openai'
import { registerAiCommands } from './ai-commands'
import { registerAiMiddleware } from './ai-middleware'
import { callAiModeration, processAiMessage, translateAiText, type TranslateTextInput } from './ai-processing'

const CONTEXT_TIMEOUT_MS = 30 * 60 * 1000
const CLEANUP_INTERVAL_MS = 10 * 60 * 1000

export interface AiLogInput {
  readonly session: Session
  readonly command: string
  readonly target: string
  readonly result: string
}

export class AIModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'ai',
    description: 'AI对话与翻译模块',
    version: '1.0.0',
    author: 'stuhelperGroupCenter',
  }

  readonly userContexts = new Map<string, UserContext>()
  readonly contextsPath: string
  readonly contextTimeout = CONTEXT_TIMEOUT_MS

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null
  private cleanupInterval: NodeJS.Timeout | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager,
    private readonly initialConfig: Config,
  ) {
    this.contextsPath = path.join(data.dataPath, 'ai_contexts.json')
  }

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
      initDataFiles(this)
      loadContexts(this)
      this.cleanupInterval = setInterval(() => cleanExpiredContexts(this), CLEANUP_INTERVAL_MS)
      registerAiCommands(this)
      registerAiMiddleware(this)
      this.data.writeLog('[ai] Module initialized')
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    saveContexts(this)
    if (this.cleanupInterval) {
      clearInterval(this.cleanupInterval)
      this.cleanupInterval = null
    }
    this.data.writeLog('[ai] Module disposed')
    this._state = 'unloaded'
  }

  registerCommand(def: RuntimeCommandDef): Command {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }

  async log(input: AiLogInput): Promise<void> {
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

  saveContexts(): void {
    saveContexts(this)
  }

  addMessageToContext(input: {
    readonly userId: string
    readonly message: ChatMessage
    readonly systemPrompt: string
    readonly contextLimit: number
  }): void {
    addMessageToContext(this, input)
  }

  callOpenAI(input: OpenAIRequestInput): Promise<ChatCompletionResponse> {
    return callOpenAI(this, input)
  }

  resetUserContext(userId: string): boolean {
    const deleted = this.userContexts.delete(userId)
    if (deleted) saveContexts(this)
    return deleted
  }

  processMessage(userId: string, content: string, guildId?: string): Promise<string> {
    return processAiMessage(this, { userId, content, guildId })
  }

  translateText(input: TranslateTextInput): Promise<string> {
    return translateAiText(this, input)
  }

  callModeration(prompt: string): Promise<string> {
    return callAiModeration(this, prompt)
  }
}

export const aiRuntimeModule: RuntimeModule<AIModule> = {
  id: 'ai',
  create(ctx, deps) {
    return new AIModule(ctx, deps.data, deps.config)
  },
}
