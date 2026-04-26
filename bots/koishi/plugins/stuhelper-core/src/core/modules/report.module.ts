import type { Command, Context, Session } from 'koishi'
import { Logger } from 'koishi'

import type { DataManager } from '../data'
import type { Config, GroupConfig, ReportGuildConfig } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { setupReportCleanupTask } from './report-cleanup'
import { registerReportConfigCommand } from './report-config-command'
import { registerReportMessageListener } from './report-context'
import { registerReportCommands } from './report-commands'
import { CONTEXT_REPORT_PROMPT, DEFAULT_REPORT_PROMPT } from './report-prompts'
import type { MessageRecord, ReportBanRecord, ReportedMessageRecord } from './report-types'
import { ViolationLevel } from './report-types'
import {
  getViolationLevelText,
  handleReportViolation,
  type ReportViolationInput,
} from './report-violation'

const logger = new Logger('stuhelperGroupCenter:report')
const DEFAULT_REPORT_COOLDOWN_MINUTES = 60
const DEFAULT_MIN_UNLIMITED_AUTHORITY = 2
const DEFAULT_MAX_REPORT_TIME_MINUTES = 30
const SECONDS_PER_MINUTE = 60
const MS_PER_SECOND = 1000
const MS_PER_MINUTE = SECONDS_PER_MINUTE * MS_PER_SECOND
const MAX_COMMAND_LOGS = 1000

export { ViolationLevel }

export class ReportModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'report',
    description: '举报模块 - AI 内容审核',
  }

  readonly reportBans: Record<string, ReportBanRecord> = {}
  readonly guildMessages: Record<string, MessageRecord[]> = {}
  readonly reportedMessages: Record<string, ReportedMessageRecord> = {}

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
      registerReportMessageListener(this)
      registerReportCommands(this)
      registerReportConfigCommand(this)
      setupReportCleanupTask(this)
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

  getReportCooldownDuration(): number {
    return (this.config.report?.maxReportCooldown || DEFAULT_REPORT_COOLDOWN_MINUTES) * MS_PER_MINUTE
  }

  getMinUnlimitedAuthority(): number {
    return this.config.report?.minAuthorityNoLimit || DEFAULT_MIN_UNLIMITED_AUTHORITY
  }

  getMaxReportTime(): number {
    return this.config.report?.maxReportTime || DEFAULT_MAX_REPORT_TIME_MINUTES
  }

  getDefaultPrompt(): string {
    return this.config.report?.defaultPrompt || DEFAULT_REPORT_PROMPT
  }

  getContextPrompt(): string {
    return this.config.report?.contextPrompt || CONTEXT_REPORT_PROMPT
  }

  getGroupConfig(guildId: string): GroupConfig | undefined {
    return this.data.groupConfig.get(guildId)
  }

  getReportGuildConfig(guildId: string): ReportGuildConfig | null {
    const groupConfig = this.getGroupConfig(guildId)
    if (groupConfig?.report) return groupConfig.report

    const reportConfig = this.config.report
    if (!reportConfig?.guildConfigs || !reportConfig.guildConfigs[guildId]) return null
    return reportConfig.guildConfigs[guildId]
  }

  async callModeration(prompt: string): Promise<string> {
    try {
      const aiModule = this.ctx.stuhelperGroupCenter.getModule<import('./ai.module').AIModule>('ai')
      if (!aiModule) throw new Error('AI 模块未加载')
      return await aiModule.callModeration(prompt)
    } catch (error) {
      logger.error('调用 AI 审核失败:', error)
      throw error
    }
  }

  handleViolation(input: Omit<ReportViolationInput, 'host'>): Promise<string> {
    return handleReportViolation({ host: this, ...input })
  }

  getViolationLevelText(level: ViolationLevel): string {
    return getViolationLevelText(level)
  }

  async logCommand(session: Session, command: string, target: string, details: string): Promise<void> {
    try {
      const commandLogs = this.data.commandLogs.getAll()
      if (!commandLogs.logs) commandLogs.logs = []

      commandLogs.logs.push({
        timestamp: Date.now(),
        guildId: session.guildId,
        userId: session.userId,
        command,
        target,
        details,
      })

      if (commandLogs.logs.length > MAX_COMMAND_LOGS) {
        commandLogs.logs = commandLogs.logs.slice(-MAX_COMMAND_LOGS)
      }

      this.data.commandLogs.set('logs', commandLogs.logs)
    } catch (error) {
      logger.error('记录命令日志失败:', error)
    }
  }
}

export const reportRuntimeModule: RuntimeModule<ReportModule> = {
  id: 'report',
  create(ctx, deps) {
    return new ReportModule(ctx, deps.data, deps.config)
  },
}
