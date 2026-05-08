import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { Config, GroupConfig, RecalledMessage } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { getRequiredPluginConfig } from './module-config'
import {
  registerAntiRecallEventListeners,
  scheduleAntiRecallCleanup,
} from './antirecall-events'
import { registerAntiRecallCommands } from './antirecall-commands'

export interface CachedMessage {
  content: string
  userId: string
  username: string
  timestamp: number
}

export interface AntiRecallLogEntry {
  session: Session
  command: string
  target: string
  result: string
  success?: boolean
}

export class AntiRecallModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'antirecall',
    description: '防撤回功能',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null
  private readonly messageCache = new Map<string, CachedMessage>()
  private cleanupInterval: ReturnType<typeof setInterval> | null = null

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
      registerAntiRecallEventListeners(this)
      registerAntiRecallCommands(this)
      scheduleAntiRecallCleanup(this)
      this.logInfo('AntiRecall module initialized')
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    this.clearCleanupInterval()
    this.messageCache.clear()
    this._state = 'unloaded'
  }

  registerCommand(def: RuntimeCommandDef) {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }

  async logCommand(entry: AntiRecallLogEntry): Promise<void> {
    if (entry.success === false) {
      entry.session['_commandFailed'] = true
    }
    await this.ctx.stuhelperGroupCenter.logCommand({
      session: entry.session,
      command: entry.command,
      target: entry.target,
      result: entry.result,
    })
  }

  logInfo(message: string): void {
    this.data.writeLog(`[antirecall] ${message}`)
  }

  getAntiRecallConfig(guildId: string): Config['antiRecall'] {
    const globalConfig = this.config.antiRecall || {}
    const groupConfigs = this.data.groupConfig.getAll()
    const groupConfig = groupConfigs[guildId]?.antiRecall || {}
    return { ...globalConfig, ...groupConfig } as Config['antiRecall']
  }

  updateGuildConfig(guildId: string, updates: Partial<Config['antiRecall']>): void {
    const groupConfigs = this.data.groupConfig.getAll()
    if (!groupConfigs[guildId]) {
      groupConfigs[guildId] = {} as GroupConfig
    }
    if (!groupConfigs[guildId].antiRecall) {
      groupConfigs[guildId].antiRecall = { enabled: false }
    }
    groupConfigs[guildId].antiRecall = {
      ...groupConfigs[guildId].antiRecall,
      ...updates,
    }
    this.data.groupConfig.setAll(groupConfigs)
  }

  isEnabledForGuild(guildId: string): boolean {
    return this.getAntiRecallConfig(guildId)?.enabled || false
  }

  getUserRecallRecords(guildId: string, userId: string, limit: number = 10): RecalledMessage[] {
    const records = this.data.recallRecords.getAll()
    return (records[guildId]?.[userId] || []).slice(0, limit)
  }

  getStatus(guildId: string) {
    const records = this.data.recallRecords.getAll()
    let totalRecords = 0
    let totalUsers = 0
    const totalGuilds = Object.keys(records).length

    Object.values(records).forEach(guildRecords => {
      const users = Object.keys(guildRecords)
      totalUsers += users.length
      users.forEach(userId => totalRecords += guildRecords[userId].length)
    })

    const effectiveConfig = this.getAntiRecallConfig(guildId)
    const globalEnabled = this.config.antiRecall?.enabled || false
    const groupSpecificEnabled = this.data.groupConfig.getAll()[guildId]?.antiRecall?.enabled

    return {
      globalEnabled,
      groupSpecificEnabled,
      effectiveConfig,
      statistics: { totalRecords, totalUsers, totalGuilds },
    }
  }

  clearAllRecords(): void {
    this.data.recallRecords.setAll({})
    this.messageCache.clear()
  }

  cacheMessage(messageId: string, message: CachedMessage): void {
    this.messageCache.set(messageId, message)
  }

  getCachedMessage(messageId: string): CachedMessage | undefined {
    return this.messageCache.get(messageId)
  }

  deleteCachedMessage(messageId: string): void {
    this.messageCache.delete(messageId)
  }

  setCleanupInterval(interval: ReturnType<typeof setInterval>): void {
    this.cleanupInterval = interval
  }

  clearCleanupInterval(): void {
    if (!this.cleanupInterval) return
    clearInterval(this.cleanupInterval)
    this.cleanupInterval = null
  }
}

export const antirecallRuntimeModule: RuntimeModule<AntiRecallModule> = {
  id: 'antirecall',
  create(ctx, deps) {
    return new AntiRecallModule(ctx, deps.data)
  },
}
