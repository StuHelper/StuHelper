import type { Context } from 'koishi'

export const GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE = 'stuhelper_group_guard_behavior_settings'
const DEFAULT_SETTINGS_ID = 'default'

declare module 'koishi' {
  interface Tables {
    [GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE]: GroupGuardBehaviorSettingsRecord
  }
}

export interface GroupGuardFunSettings {
  diceSides: number
  muteLotteryBaseSeconds: number
  muteLotteryMaxSeconds: number
  muteLotteryPityThreshold: number
  muteLotteryPitySeconds: number
}

export interface GroupGuardModerationSettings {
  repeatThreshold: number
  repeatWindowSize: number
  warningThresholdExpression: string
  defaultMuteSeconds: number
  antiRecallNotify: boolean
}

export interface GroupGuardBehaviorSettings {
  fun: GroupGuardFunSettings
  moderation: GroupGuardModerationSettings
}

export interface GroupGuardBehaviorSettingsRecord extends GroupGuardBehaviorSettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export interface GroupGuardBehaviorSettingsInput {
  fun?: Partial<GroupGuardFunSettings>
  moderation?: Partial<GroupGuardModerationSettings>
}

export const DEFAULT_GROUP_GUARD_FUN_SETTINGS: GroupGuardFunSettings = Object.freeze({
  diceSides: 100,
  muteLotteryBaseSeconds: 120,
  muteLotteryMaxSeconds: 600,
  muteLotteryPityThreshold: 5,
  muteLotteryPitySeconds: 300,
})

export const DEFAULT_GROUP_GUARD_MODERATION_SETTINGS: GroupGuardModerationSettings = Object.freeze({
  repeatThreshold: 3,
  repeatWindowSize: 3,
  warningThresholdExpression: 'warnings >= 3',
  defaultMuteSeconds: 600,
  antiRecallNotify: true,
})

export const DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS: GroupGuardBehaviorSettings = Object.freeze({
  fun: DEFAULT_GROUP_GUARD_FUN_SETTINGS,
  moderation: DEFAULT_GROUP_GUARD_MODERATION_SETTINGS,
})

export function registerGroupGuardBehaviorSettingsModel(ctx: Context) {
  ctx.model.extend(GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE, {
    id: 'string',
    fun: { type: 'json', initial: null },
    moderation: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class GroupGuardBehaviorSettingsStore {
  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: GroupGuardBehaviorSettings = DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
  ) {}

  async getSettings(): Promise<GroupGuardBehaviorSettingsRecord> {
    const [record] = await this.ctx.database.get(GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
    if (record) {
      return normalizeRecord(record, this.defaults)
    }
    return this.createDefaultRecord()
  }

  async saveSettings(input: GroupGuardBehaviorSettingsInput) {
    const current = await this.getSettings()
    const changes = normalizeInput(input, current, this.defaults)
    const now = new Date()
    const next: GroupGuardBehaviorSettingsRecord = {
      ...current,
      ...changes,
      updatedAt: now,
    }
    await this.ctx.database.set(GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return next
  }

  async resetSettings() {
    const current = await this.getSettings()
    const now = new Date()
    const next: GroupGuardBehaviorSettingsRecord = {
      ...current,
      fun: normalizeFunSettings(this.defaults.fun, DEFAULT_GROUP_GUARD_FUN_SETTINGS),
      moderation: normalizeModerationSettings(this.defaults.moderation, DEFAULT_GROUP_GUARD_MODERATION_SETTINGS),
      updatedAt: now,
    }
    await this.ctx.database.set(GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      fun: next.fun,
      moderation: next.moderation,
      updatedAt: now,
    })
    return next
  }

  async getFunSettings() {
    return (await this.getSettings()).fun
  }

  async getModerationSettings() {
    return (await this.getSettings()).moderation
  }

  private async createDefaultRecord() {
    const now = new Date()
    const record: GroupGuardBehaviorSettingsRecord = {
      id: DEFAULT_SETTINGS_ID,
      fun: normalizeFunSettings(this.defaults.fun, DEFAULT_GROUP_GUARD_FUN_SETTINGS),
      moderation: normalizeModerationSettings(this.defaults.moderation, DEFAULT_GROUP_GUARD_MODERATION_SETTINGS),
      createdAt: now,
      updatedAt: now,
    }
    try {
      await this.ctx.database.create(GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE, record)
    } catch (error) {
      const [existing] = await this.ctx.database.get(GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
      if (existing) {
        return normalizeRecord(existing, this.defaults)
      }
      throw error
    }
    return record
  }
}

function normalizeRecord(
  record: GroupGuardBehaviorSettingsRecord,
  defaults: GroupGuardBehaviorSettings,
): GroupGuardBehaviorSettingsRecord {
  return {
    ...record,
    fun: normalizeFunSettings(record.fun, defaults.fun),
    moderation: normalizeModerationSettings(record.moderation, defaults.moderation),
  }
}

function normalizeInput(
  input: GroupGuardBehaviorSettingsInput,
  current: GroupGuardBehaviorSettings,
  defaults: GroupGuardBehaviorSettings,
): Partial<GroupGuardBehaviorSettings> {
  const changes: Partial<GroupGuardBehaviorSettings> = {}
  const fun = normalizePartialFunSettings(input.fun)
  if (fun) {
    changes.fun = normalizeFunSettings({
      ...defaults.fun,
      ...current.fun,
      ...fun,
    }, defaults.fun)
  }
  const moderation = normalizePartialModerationSettings(input.moderation)
  if (moderation) {
    changes.moderation = normalizeModerationSettings({
      ...defaults.moderation,
      ...current.moderation,
      ...moderation,
    }, defaults.moderation)
  }
  return changes
}

function normalizeFunSettings(value: unknown, fallback: GroupGuardFunSettings): GroupGuardFunSettings {
  const record = isObject(value) ? value : {}
  return {
    diceSides: positiveIntegerOrDefault(record.diceSides, fallback.diceSides),
    muteLotteryBaseSeconds: positiveIntegerOrDefault(record.muteLotteryBaseSeconds, fallback.muteLotteryBaseSeconds),
    muteLotteryMaxSeconds: positiveIntegerOrDefault(record.muteLotteryMaxSeconds, fallback.muteLotteryMaxSeconds),
    muteLotteryPityThreshold: positiveIntegerOrDefault(record.muteLotteryPityThreshold, fallback.muteLotteryPityThreshold),
    muteLotteryPitySeconds: positiveIntegerOrDefault(record.muteLotteryPitySeconds, fallback.muteLotteryPitySeconds),
  }
}

function normalizePartialFunSettings(value: unknown) {
  if (!isObject(value)) {
    return undefined
  }
  const result: Partial<GroupGuardFunSettings> = {}
  for (const key of GROUP_GUARD_FUN_SETTING_KEYS) {
    const setting = value[key]
    if (isPositiveInteger(setting)) {
      result[key] = setting
    }
  }
  return Object.keys(result).length ? result : undefined
}

function normalizeModerationSettings(value: unknown, fallback: GroupGuardModerationSettings): GroupGuardModerationSettings {
  const record = isObject(value) ? value : {}
  return {
    repeatThreshold: positiveIntegerOrDefault(record.repeatThreshold, fallback.repeatThreshold),
    repeatWindowSize: positiveIntegerOrDefault(record.repeatWindowSize, fallback.repeatWindowSize),
    warningThresholdExpression: nonEmptyStringOrDefault(record.warningThresholdExpression, fallback.warningThresholdExpression),
    defaultMuteSeconds: positiveIntegerOrDefault(record.defaultMuteSeconds, fallback.defaultMuteSeconds),
    antiRecallNotify: booleanOrDefault(record.antiRecallNotify, fallback.antiRecallNotify),
  }
}

function normalizePartialModerationSettings(value: unknown) {
  if (!isObject(value)) {
    return undefined
  }
  const result: Partial<GroupGuardModerationSettings> = {}
  for (const key of GROUP_GUARD_MODERATION_SETTING_KEYS) {
    const setting = value[key]
    if (key === 'warningThresholdExpression') {
      if (typeof setting === 'string' && setting.trim()) {
        result[key] = setting.trim()
      }
      continue
    }
    if (key === 'antiRecallNotify') {
      if (typeof setting === 'boolean') {
        result[key] = setting
      }
      continue
    }
    if (isPositiveInteger(setting)) {
      result[key] = setting
    }
  }
  return Object.keys(result).length ? result : undefined
}

export const GROUP_GUARD_FUN_SETTING_KEYS = Object.freeze([
  'diceSides',
  'muteLotteryBaseSeconds',
  'muteLotteryMaxSeconds',
  'muteLotteryPityThreshold',
  'muteLotteryPitySeconds',
] as const)

export const GROUP_GUARD_MODERATION_SETTING_KEYS = Object.freeze([
  'repeatThreshold',
  'repeatWindowSize',
  'warningThresholdExpression',
  'defaultMuteSeconds',
  'antiRecallNotify',
] as const)

function positiveIntegerOrDefault(value: unknown, fallback: number) {
  return isPositiveInteger(value) ? value : fallback
}

function isPositiveInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value > 0
}

function nonEmptyStringOrDefault(value: unknown, fallback: string) {
  return typeof value === 'string' && value.trim() ? value.trim() : fallback
}

function booleanOrDefault(value: unknown, fallback: boolean) {
  return typeof value === 'boolean' ? value : fallback
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
