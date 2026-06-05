import type { Session } from 'koishi'

import type { DataManager } from '../data'
import type { BanMeConfig, Config, GroupConfig } from '../../types'
import type { RuntimeCommand, RuntimeCommandDef } from '../../runtime/types'
import type { SimilarChars } from './banme-similar-chars'
import { parseTimeString } from '../../utils'

const ENABLED_VALUES = new Set(['true', '1', 'yes', 'y', 'on'])
const DISABLED_VALUES = new Set(['false', '0', 'no', 'n', 'off'])

interface BanmeConfigCommandOptions {
  readonly enabled?: unknown
  readonly baseMin?: unknown
  readonly baseMax?: unknown
  readonly rate?: unknown
  readonly prob?: unknown
  readonly spity?: unknown
  readonly hpity?: unknown
  readonly uptime?: unknown
  readonly losetime?: unknown
  readonly autoBan?: unknown
  readonly reset?: unknown
}

export interface BanmeCommandHost {
  readonly data: DataManager
  readonly config: Config
  registerCommand(def: RuntimeCommandDef): RuntimeCommand
  executeBanme(session: Session): Promise<string | null>
  normalizeCommand(command: string): string
  readSimilarChars(): SimilarChars | null
  saveSimilarChars(data: SimilarChars): void
  setDefaultSimilarChars(): void
  log(entry: {
    readonly session: Session
    readonly command: string
    readonly target: string
    readonly result: string
  }): Promise<void>
}

export function registerBanmeCommands(host: BanmeCommandHost): void {
  registerMainCommand(host)
  registerSimilarCommand(host)
  registerNormalizeCommand(host)
  registerRecordCommand(host)
  registerAliasCommand(host)
  registerConfigCommand(host)
}

function registerMainCommand(host: BanmeCommandHost): void {
  host.registerCommand({
    name: 'banme',
    desc: '随机禁言自己',
    skipAuth: true,
    usage: '随机抽取禁言时长，支持抽卡保底系统',
  }).action(async ({ session }) => {
    if (!session) return '无法读取当前会话'
    if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
    if (session.quote) return '喵喵？回复消息时不能使用这个命令哦~'
    return host.executeBanme(session)
  })
}

function registerSimilarCommand(host: BanmeCommandHost): void {
  host.registerCommand({
    name: 'banme.similar',
    desc: '查看形似字符映射表',
    permDesc: '查看 banme 形似字符映射配置',
    usage: '显示当前配置的形似字符替换规则',
  }).action(() => {
    const similarChars = host.readSimilarChars()
    if (!similarChars || Object.keys(similarChars).length === 0) {
      host.setDefaultSimilarChars()
      return '没有找到 banme 形似字符映射，已设置默认映射喵~'
    }

    const charList = Object.entries(similarChars)
      .map(([char, replacement]) => `${char} -> ${replacement}`)
      .join('\n')
    return `当前的 banme 形似字符映射如下喵~\n${charList || '没有形似字符映射喵~'}`
  })
}

function registerNormalizeCommand(host: BanmeCommandHost): void {
  host.registerCommand({
    name: 'banme.normalize',
    desc: '规范化命令测试',
    args: '<command:string>',
    permDesc: '测试 banme 规范化功能',
    usage: '测试字符串规范化结果，用于调试形似字符',
    examples: ['banme.normalize bаnmе'],
  }).action(({ session }, command) => {
    if (!session) return '无法读取当前会话'
    if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

    const normalizedCommand = host.normalizeCommand(host.normalizeCommand(command))
    const charList = normalizedCommand
      .split('')
      .map((char, index) => `${index + 1}. ${char.charCodeAt(0).toString(16)}`)
      .join('\n')
    return `规范化后的命令：${normalizedCommand}\n长度：${normalizedCommand.length}\n字符列表：\n${charList}`
  })
}

function registerRecordCommand(host: BanmeCommandHost): void {
  host.registerCommand({
    name: 'banme.record',
    desc: '记录形似字符映射',
    args: '<standardCommand:string>',
    permDesc: '通过引用消息逐字符添加形似字符替换',
    usage: '引用一条包含形似字符的消息，提供标准字符串进行映射',
  }).action(async ({ session }, standardCommand) => {
    if (!session) return '无法读取当前会话'
    if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
    if (!session.quote) return '请引用一条消息来记录映射喵~'
    if (typeof standardCommand !== 'string' || standardCommand.length === 0) return '请提供标准命令字符串喵~'

    const quotedMessage = session.quote.content
    const normalizedCommand = host.normalizeCommand(host.normalizeCommand(quotedMessage))

    if (normalizedCommand.length !== standardCommand.length) {
      return '映射记录失败喵~\n' + '规范化字符串:' + normalizedCommand + '\n' + '对应的标准串:' + standardCommand + '\n' + '两者长度不一致喵~'
    }

    const similarChars = host.readSimilarChars() || {}
    for (let i = 0; i < normalizedCommand.length; i++) {
      const originalChar = normalizedCommand[i]
      const standardChar = standardCommand[i]
      if (standardChar !== originalChar) {
        similarChars[originalChar] = standardChar
      }
    }

    host.saveSimilarChars(similarChars)
    void host.log({ session, command: 'banme.record', target: session.userId, result: '成功' })
    return '已记录形似字符映射喵~\n' + '规范化字符串：' + normalizedCommand + '\n' + '对应的标准串：' + standardCommand
  })
}

function registerAliasCommand(host: BanmeCommandHost): void {
  host.registerCommand({
    name: 'banme.alias',
    desc: '记录字符串别名',
    args: '<standardCommand:string>',
    permDesc: '通过引用消息添加字符串映射',
    usage: '引用一条消息，将其整体映射为标准字符串',
  }).action(async ({ session }, standardCommand) => {
    if (!session) return '无法读取当前会话'
    if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
    if (!session.quote) return '请引用一条消息来记录映射喵~'
    if (typeof standardCommand !== 'string' || standardCommand.length === 0) return '请提供一个标准字符串喵~'

    const quotedMessage = session.quote.content
    const similarChars = host.readSimilarChars() || {}
    similarChars[quotedMessage] = standardCommand

    host.saveSimilarChars(similarChars)
    void host.log({ session, command: 'banme.alias', target: session.userId, result: '成功' })
    return '已记录字符串映射喵~\n' + '原字符串：' + quotedMessage + '\n' + '对应的标准串：' + standardCommand
  })
}

function registerConfigCommand(host: BanmeCommandHost): void {
  host.registerCommand({
    name: 'banme.config',
    desc: '设置banme配置',
    permDesc: '修改 banme 功能配置',
    usage: '配置本群的 banme 参数，包括启用、时长、概率等',
  })
    .option('enabled', '--enabled <enabled:boolean> 是否启用')
    .option('baseMin', '--baseMin <seconds:number> 最小禁言时间(秒)')
    .option('baseMax', '--baseMax <minutes:number> 最大禁言时间(分)')
    .option('rate', '--rate <rate:number> 增长率')
    .option('prob', '--prob <probability:number> 金卡基础概率')
    .option('spity', '--spity <count:number> 软保底抽数')
    .option('hpity', '--hpity <count:number> 硬保底抽数')
    .option('uptime', '--uptime <duration:string> UP奖励时长')
    .option('losetime', '--losetime <duration:string> 歪奖励时长')
    .option('autoBan', '--autoBan <enabled:boolean> 是否自动禁言使用特殊字符的用户')
    .option('reset', '--reset 重置为全局配置')
    .action(async ({ session, options }) => {
      if (!session) return '无法读取当前会话'
      const guildId = session.guildId
      if (!guildId) return '喵呜...这个命令只能在群里用喵...'
      return handleConfigCommand(host, session, guildId, normalizeConfigOptions(options))
    })
}

function handleConfigCommand(
  host: BanmeCommandHost,
  session: Session,
  guildId: string,
  options: BanmeConfigCommandOptions,
): string {
  const currentConfigs = host.data.groupConfig.getAll() as Record<string, GroupConfig>
  const currentGroupConfig = currentConfigs[guildId] || {}

  if (isResetRequested(options.reset)) {
    const nextConfigs = {
      ...currentConfigs,
      [guildId]: { ...currentGroupConfig },
    }
    delete nextConfigs[guildId].banme
    host.data.groupConfig.setAll(nextConfigs)
    return '已重置为全局配置喵~'
  }

  const banmeConfig = cloneBanmeConfig(currentGroupConfig.banme || host.config.banme)

  const error = applyBooleanOptions({ host, session, config: banmeConfig, options })
  if (error) return error

  const numericError = applyNumericOptions({ host, session, config: banmeConfig, options })
  if (numericError) return numericError

  const durationError = applyDurationOptions({ host, session, config: banmeConfig, options })
  if (durationError) return durationError

  const rangeError = validateBanmeConfigRange(host, session, banmeConfig)
  if (rangeError) return rangeError

  host.data.groupConfig.setAll({
    ...currentConfigs,
    [guildId]: {
      ...currentGroupConfig,
      banme: banmeConfig,
    },
  })
  void host.log({ session, command: 'banme.config', target: session.userId, result: '成功：更新banme配置' })
  return '配置已更新喵~'
}

function applyBooleanOptions(input: {
  readonly host: BanmeCommandHost
  readonly session: Session
  readonly config: BanMeConfig
  readonly options: BanmeConfigCommandOptions
}): string | undefined {
  const { host, session, config, options } = input
  const enabled = parseBooleanOption(options.enabled)
  if (options.enabled !== undefined && enabled === undefined) {
    void host.log({ session, command: 'banme.config', target: session.userId, result: '失败：启用选项无效' })
    return '启用选项无效，请输入 true/false'
  }
  if (enabled !== undefined) config.enabled = enabled

  const autoBan = parseBooleanOption(options.autoBan)
  if (options.autoBan !== undefined && autoBan === undefined) {
    void host.log({ session, command: 'banme.config', target: session.userId, result: '失败：自动禁言选项无效' })
    return '自动禁言选项无效，请输入 true/false'
  }
  if (autoBan !== undefined) config.autoBan = autoBan
  return undefined
}

function applyNumericOptions(input: {
  readonly host: BanmeCommandHost
  readonly session: Session
  readonly config: BanMeConfig
  readonly options: BanmeConfigCommandOptions
}): string | undefined {
  const { host, session, config, options } = input
  const baseMin = readPositiveNumberOption(options.baseMin, '最小禁言时间')
  if (typeof baseMin === 'string') return logConfigValidationFailure(host, session, baseMin)
  if (baseMin !== undefined) config.baseMin = baseMin

  const baseMax = readPositiveNumberOption(options.baseMax, '最大禁言时间')
  if (typeof baseMax === 'string') return logConfigValidationFailure(host, session, baseMax)
  if (baseMax !== undefined) config.baseMax = baseMax

  const rate = readNonNegativeNumberOption(options.rate, '增长率')
  if (typeof rate === 'string') return logConfigValidationFailure(host, session, rate)
  if (rate !== undefined) config.growthRate = rate

  const probability = readProbabilityOption(options.prob)
  if (typeof probability === 'string') return logConfigValidationFailure(host, session, probability)
  if (probability !== undefined) config.jackpot.baseProb = probability

  const softPity = readPositiveIntegerOption(options.spity, '软保底抽数')
  if (typeof softPity === 'string') return logConfigValidationFailure(host, session, softPity)
  if (softPity !== undefined) config.jackpot.softPity = softPity

  const hardPity = readPositiveIntegerOption(options.hpity, '硬保底抽数')
  if (typeof hardPity === 'string') return logConfigValidationFailure(host, session, hardPity)
  if (hardPity !== undefined) config.jackpot.hardPity = hardPity

  return undefined
}

function applyDurationOptions(input: {
  readonly host: BanmeCommandHost
  readonly session: Session
  readonly config: BanMeConfig
  readonly options: BanmeConfigCommandOptions
}): string | undefined {
  const { host, session, config, options } = input
  const upDuration = readDurationOption(options.uptime, 'UP奖励时长')
  if (upDuration.error) return logConfigValidationFailure(host, session, upDuration.error)
  if (upDuration.value !== undefined) config.jackpot.upDuration = upDuration.value

  const loseDuration = readDurationOption(options.losetime, '歪奖励时长')
  if (loseDuration.error) return logConfigValidationFailure(host, session, loseDuration.error)
  if (loseDuration.value !== undefined) config.jackpot.loseDuration = loseDuration.value
  return undefined
}

function normalizeConfigOptions(value: unknown): BanmeConfigCommandOptions {
  if (!isRecord(value)) return {}
  return {
    enabled: value.enabled,
    baseMin: value.baseMin,
    baseMax: value.baseMax,
    rate: value.rate,
    prob: value.prob,
    spity: value.spity,
    hpity: value.hpity,
    uptime: value.uptime,
    losetime: value.losetime,
    autoBan: value.autoBan,
    reset: value.reset,
  }
}

function parseBooleanOption(value: unknown): boolean | undefined {
  if (value === undefined) return undefined

  const normalized = String(value).trim().toLowerCase()
  if (ENABLED_VALUES.has(normalized)) return true
  if (DISABLED_VALUES.has(normalized)) return false
  return undefined
}

function cloneBanmeConfig(config: BanMeConfig): BanMeConfig {
  return {
    ...config,
    jackpot: {
      ...config.jackpot,
    },
  }
}

function readPositiveNumberOption(value: unknown, label: string): number | string | undefined {
  if (value === undefined) return undefined
  const numberValue = normalizeNumber(value)
  if (numberValue === null || numberValue <= 0) return `${label}必须是正数`
  return numberValue
}

function readNonNegativeNumberOption(value: unknown, label: string): number | string | undefined {
  if (value === undefined) return undefined
  const numberValue = normalizeNumber(value)
  if (numberValue === null || numberValue < 0) return `${label}必须是非负数`
  return numberValue
}

function readProbabilityOption(value: unknown): number | string | undefined {
  if (value === undefined) return undefined
  const numberValue = normalizeNumber(value)
  if (numberValue === null || numberValue < 0 || numberValue > 1) return '金卡基础概率必须是 0 到 1 之间的数字'
  return numberValue
}

function readPositiveIntegerOption(value: unknown, label: string): number | string | undefined {
  if (value === undefined) return undefined
  const numberValue = normalizeNumber(value)
  if (numberValue === null || !Number.isInteger(numberValue) || numberValue <= 0) return `${label}必须是正整数`
  return numberValue
}

function readDurationOption(value: unknown, label: string): { value?: string; error?: string } {
  if (value === undefined) return {}
  if (typeof value !== 'string' || !value.trim()) return { error: `${label}格式无效` }

  const duration = value.trim()
  if (!isDurationOptionValid(duration)) return { error: `${label}格式无效` }
  return { value: duration }
}

function validateBanmeConfigRange(
  host: BanmeCommandHost,
  session: Session,
  config: BanMeConfig,
): string | undefined {
  if (config.baseMin > config.baseMax * 60) {
    return logConfigValidationFailure(host, session, '最小禁言时间不能大于最大禁言时间')
  }
  if (config.jackpot.softPity > config.jackpot.hardPity) {
    return logConfigValidationFailure(host, session, '软保底抽数不能大于硬保底抽数')
  }
  return undefined
}

function logConfigValidationFailure(
  host: BanmeCommandHost,
  session: Session,
  message: string,
): string {
  void host.log({ session, command: 'banme.config', target: session.userId, result: `失败：${message}` })
  return message
}

function normalizeNumber(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null
  return value
}

function isDurationOptionValid(duration: string): boolean {
  try {
    parseTimeString(duration)
    return true
  } catch {
    return false
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isResetRequested(value: unknown): boolean {
  return value === true
}
