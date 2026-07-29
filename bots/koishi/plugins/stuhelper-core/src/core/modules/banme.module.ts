import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { BanMeConfig, BanMeRecord, Config, GroupConfig } from '../../types'
import { parseTimeString, formatDuration } from '../../utils'
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
import { registerBanmeCommands } from './banme-commands'
import {
  normalizeBanmeCommand,
  SimilarCharsStore,
  type SimilarChars,
} from './banme-similar-chars'
import { formatBanmeMessage, formatSuccessLog } from './banme-messages'
import type { JackpotResult } from './banme-types'
import { commandErrorMessage } from './command-error-message'
import { secureRandomInt, secureRandomUnit } from '../../utils/secure-random'

const SIMILAR_CHARS_PATH = './data/similarChars.json'
const HOUR_MS = 3_600_000
const MAX_MUTE_MS = 30 * 24 * 60 * 60 * 1000
const DEFAULT_BASE_PROBABILITY = 0.006
const DEFAULT_SOFT_PITY = 74
const DEFAULT_HARD_PITY = 90
const SOFT_PITY_STEP = 0.06

/**
 * 自助禁言模块
 * 支持抽卡系统和形似字符检测
 */
export class BanmeModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'banme',
    description: '自助禁言模块',
    version: '1.0.0',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null
  private readonly similarChars: SimilarCharsStore

  constructor(
    private readonly ctx: Context,
    private readonly _data: DataManager
  ) {
    this.similarChars = new SimilarCharsStore(SIMILAR_CHARS_PATH, this.ctx.logger)
  }

  get data(): DataManager {
    return this._data
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
      this.similarChars.ensure()
      this.registerMiddleware()
      registerBanmeCommands(this)
      this.ctx.logger.info('[BanmeModule] initialized')
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

  readSimilarChars(): SimilarChars | null {
    return this.similarChars.read()
  }

  saveSimilarChars(data: SimilarChars): void {
    this.similarChars.save(data)
  }

  setDefaultSimilarChars(): void {
    this.similarChars.writeDefaults()
  }

  normalizeCommand(command: string): string {
    let similarChars = this.similarChars.read()
    if (!similarChars || Object.keys(similarChars).length === 0) {
      this.similarChars.writeDefaults()
      similarChars = this.similarChars.read()
    }
    if (!similarChars) throw new Error('Banme similar chars are unavailable.')
    return normalizeBanmeCommand(command, similarChars)
  }

  async log(entry: {
    readonly session: Session
    readonly command: string
    readonly target: string
    readonly result: string
  }): Promise<void> {
    await this.ctx.stuhelperGroupCenter.logCommand(entry)
  }

  async executeBanme(session: Session, isAuto = false): Promise<string | null> {
    const guildId = session.guildId!
    const banmeConfig = this.getBanmeConfig(guildId)

    if (!banmeConfig?.enabled) {
      void this.log({ session, command: 'banme', target: session.userId, result: '失败：功能禁用' })
      return '喵呜...banme功能现在被禁用了呢...'
    }

    try {
      const records = this.data.banmeRecords.getAll()
      const now = Date.now()
      const record = getActiveBanmeRecord(records, guildId, now)
      const jackpot = rollJackpot(record, banmeConfig)
      this.data.banmeRecords.setAll(records)

      const milliseconds = calculateMuteDuration(record, banmeConfig, jackpot.isJackpot)
      await session.bot.muteGuildMember(guildId, session.userId, milliseconds)
      this.saveMuteRecord({ guildId, userId: session.userId, now, milliseconds })

      const timeStr = formatDuration(milliseconds)
      void this.log({ session, command: 'banme', target: session.userId, result: formatSuccessLog(timeStr, jackpot, record) })
      return formatBanmeMessage({ session, isAuto, timeStr, jackpot, record })
    } catch (error) {
      const message = commandErrorMessage(error)
      void this.log({ session, command: 'banme', target: session.userId, result: '失败：未知错误' })
      return `喵呜...禁言失败了：${message}`
    }
  }

  private registerMiddleware(): void {
    this.ctx.middleware(async (session, next) => {
      if (!session.content || !session.guildId) return next()

      const normalizedContent = this.normalizeCommand(this.normalizeCommand(session.content))
      if (normalizedContent !== 'banme') return next()

      if (session.content !== 'banme' && this.getBanmeConfig(session.guildId)?.autoBan) {
        try {
          const result = await this.executeBanme(session, true)
          if (result) {
            await session.send(result)
            return
          }
        } catch {
          await session.send('自动禁言失败了...可能是权限不够喵')
          return
        }
      }

      session.content = 'banme'
      return next()
    })
  }

  private getGroupConfig(guildId: string): GroupConfig | undefined {
    return this.data.groupConfig.get(guildId)
  }

  private getBanmeConfig(guildId: string): BanMeConfig {
    return this.getGroupConfig(guildId)?.banme || this.config.banme
  }

  private saveMuteRecord(input: {
    readonly guildId: string
    readonly userId: string
    readonly now: number
    readonly milliseconds: number
  }): void {
    const { guildId, userId, now, milliseconds } = input
    const allMutes = this.data.mutes.getAll()
    allMutes[guildId] = allMutes[guildId] || {}
    allMutes[guildId][userId] = { startTime: now, duration: milliseconds }
    this.data.mutes.setAll(allMutes)
  }
}

export const banmeRuntimeModule: RuntimeModule<BanmeModule> = {
  id: 'banme',
  create(ctx, deps) {
    return new BanmeModule(ctx, deps.data)
  },
}

function getActiveBanmeRecord(
  records: Record<string, BanMeRecord>,
  guildId: string,
  now: number,
): BanMeRecord {
  records[guildId] = records[guildId] || {
    count: 0,
    lastResetTime: now,
    pity: 0,
    guaranteed: false,
  }

  if (now - records[guildId].lastResetTime > HOUR_MS) {
    records[guildId].count = 0
    records[guildId].lastResetTime = now
  }

  records[guildId].count += 1
  records[guildId].pity += 1
  return records[guildId]
}

function rollJackpot(record: BanMeRecord, config: BanMeConfig): JackpotResult {
  const hardPity = config.jackpot?.hardPity || DEFAULT_HARD_PITY
  const currentProb = calculateCurrentProbability(record.pity, config)

  if (record.pity < hardPity && secureRandomUnit() >= currentProb) {
    return { isJackpot: false, isGuaranteed: false }
  }

  const isGuaranteed = record.pity >= hardPity
  record.pity = 0
  if (record.guaranteed) {
    record.guaranteed = false
  } else if (secureRandomUnit() < 0.5) {
    record.guaranteed = true
  }
  return { isJackpot: true, isGuaranteed }
}

function calculateCurrentProbability(pity: number, config: BanMeConfig): number {
  const baseProbability = config.jackpot?.baseProb || DEFAULT_BASE_PROBABILITY
  const softPity = config.jackpot?.softPity || DEFAULT_SOFT_PITY
  if (pity < softPity) return baseProbability
  return baseProbability + (pity - softPity + 1) * SOFT_PITY_STEP
}

function calculateMuteDuration(
  record: BanMeRecord,
  config: BanMeConfig,
  isJackpot: boolean,
): number {
  if (isJackpot && config.jackpot?.enabled) {
    const duration = record.guaranteed
      ? config.jackpot.loseDuration || '1d'
      : config.jackpot.upDuration || '7d'
    return parseTimeString(duration)
  }

  const baseMax = requireBanmeNumber(config.baseMax, 'baseMax')
  const baseMin = requireBanmeNumber(config.baseMin, 'baseMin')
  const growthRate = requireBanmeNumber(config.growthRate, 'growthRate')
  const baseMaxMillis = baseMax * 60 * 1000
  const baseMinMillis = Math.max(baseMin * 1000, 1000)
  const additionalMinutes = Math.floor(Math.pow(record.count - 1, 1 / 3) * growthRate)
  const maxMilliseconds = Math.min(baseMaxMillis + (additionalMinutes * 60 * 1000), MAX_MUTE_MS)
  return secureRandomInt(baseMinMillis, maxMilliseconds)
}

function requireBanmeNumber(value: number | undefined, field: string): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`banme.${field} 配置缺失或不是数字`)
  }
  return value
}
