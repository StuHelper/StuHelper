/**
 * 日志模块 - 操作日志和命令日志管理
 */
import type { Context, Session } from 'koishi'
import * as fs from 'fs'
import * as path from 'path'

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
import { registerCommandLogCommands } from './log-command-commands'
import { registerLogEventListeners } from './log-events'
import { registerOperationLogCommands } from './log-operation-commands'

const JSON_INDENT_SPACES = 2
const DEFAULT_RECENT_LOG_LIMIT = 100

export interface CommandLogRecord {
  id: string
  timestamp: string
  userId: string
  username?: string
  userAuthority?: number
  guildId?: string
  guildName?: string
  channelId?: string
  platform: string
  command: string
  args: string[]
  options: Record<string, any>
  success: boolean
  error?: string
  executionTime: number
  result?: string
  messageId?: string
  isPrivate: boolean
}

export interface LogCommandEntry {
  session: Session
  command: string
  target: string
  result: string
  success?: boolean
}

export class LogModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'log',
    description: '日志管理模块',
    version: '1.0.0',
  }

  readonly logPath: string
  readonly commandLogPath: string
  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null
  private readonly commandStats = new Map<string, { count: number, lastUsed: number }>()

  constructor(
    readonly ctx: Context,
    readonly data: DataManager
  ) {
    this.logPath = path.resolve(this.data.dataPath, 'stuhelperGroupCenter.log')
    this.commandLogPath = path.resolve(this.data.dataPath, 'command_logs.json')
  }

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
      this.initCommandLogs()
      registerOperationLogCommands(this)
      registerCommandLogCommands(this)
      registerLogEventListeners(this)
      console.log(`[${this.meta.name}] LogModule initialized`)
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

  async logCommand(entry: LogCommandEntry): Promise<void> {
    if (entry.success === false) {
      entry.session['_commandFailed'] = true
    }
    await this.ctx.stuhelperGroupCenter.logCommand(
      entry.session,
      entry.command,
      entry.target,
      entry.result,
    )
  }

  readCommandLogs(): CommandLogRecord[] {
    if (!fs.existsSync(this.commandLogPath)) {
      return []
    }
    const content = fs.readFileSync(this.commandLogPath, 'utf-8')
    const logs = JSON.parse(content)
    if (!Array.isArray(logs)) {
      throw new Error(`命令日志文件格式无效: ${this.commandLogPath}`)
    }
    return logs
  }

  saveCommandLogs(logs: CommandLogRecord[]): void {
    fs.writeFileSync(this.commandLogPath, JSON.stringify(logs, null, JSON_INDENT_SPACES), 'utf-8')
  }

  loadStats(): void {
    this.commandStats.clear()
    this.readCommandLogs().forEach(log => {
      const stats = this.commandStats.get(log.command) || { count: 0, lastUsed: 0 }
      stats.count++
      const logTime = new Date(log.timestamp).getTime()
      if (logTime > stats.lastUsed) {
        stats.lastUsed = logTime
      }
      this.commandStats.set(log.command, stats)
    })
  }

  clearCommandStats(): void {
    this.commandStats.clear()
  }

  recordCommandUsage(commandName: string): void {
    const stats = this.commandStats.get(commandName) || { count: 0, lastUsed: 0 }
    stats.count++
    stats.lastUsed = Date.now()
    this.commandStats.set(commandName, stats)
  }

  getCommandStats(): Map<string, { count: number, lastUsed: number }> {
    return new Map(this.commandStats)
  }

  async getRecentLogs(limit: number = DEFAULT_RECENT_LOG_LIMIT): Promise<CommandLogRecord[]> {
    return this.readCommandLogs().slice(-limit).reverse()
  }

  async getAllLogs(): Promise<CommandLogRecord[]> {
    return this.readCommandLogs().reverse()
  }

  private initCommandLogs(): void {
    if (!fs.existsSync(this.commandLogPath)) {
      this.saveCommandLogs([])
    }
    this.loadStats()
  }
}

export const logRuntimeModule: RuntimeModule<LogModule> = {
  id: 'log',
  create(ctx, deps) {
    return new LogModule(ctx, deps.data)
  },
}
