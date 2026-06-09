import {
  DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
  GROUP_GUARD_FUN_SETTING_KEYS,
  GROUP_GUARD_MODERATION_SETTING_KEYS,
  GroupGuardBehaviorSettingsStore,
  type GroupGuardBehaviorSettingsInput,
} from '@stuhelper/koishi-shared'

import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

type PlainRecord = Record<string, unknown>

const MAX_DICE_SIDES = 1000
const MAX_MUTE_SECONDS = 2_592_000
const MAX_PITY_THRESHOLD = 100_000
const MAX_REPEAT_LIMIT = 10_000
const MAX_WARNING_EXPRESSION_LENGTH = 256

export function registerGroupGuardBehaviorSettingsAPI(api: WebSocketAPIContext): void {
  const store = new GroupGuardBehaviorSettingsStore(api.ctx, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS)

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-behavior-settings/get', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard behavior settings')
      return success(await store.getSettings())
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '获取群管行为设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-behavior-settings/update', async function (params?: { settings?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard behavior settings')
      const settings = parseGroupGuardBehaviorSettingsInput(params?.settings)
      return success(await store.saveSettings(settings))
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '保存群管行为设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-behavior-settings/reset', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard behavior settings')
      return success(await store.resetSettings())
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '重置群管行为设置失败')
    }
  })
}

export function parseGroupGuardBehaviorSettingsInput(input: unknown): GroupGuardBehaviorSettingsInput {
  const record = requireRecord(input, 'group guard behavior settings')
  const result: GroupGuardBehaviorSettingsInput = {}

  for (const key of Object.keys(record)) {
    if (key !== 'fun' && key !== 'moderation') {
      throw new Error(`group guard behavior settings contains unsupported field: ${key}`)
    }
  }

  if ('fun' in record) {
    const fun = requireRecord(record.fun, 'group guard fun settings')
    const parsedFun: NonNullable<GroupGuardBehaviorSettingsInput['fun']> = {}
    const parsedFunRecord = parsedFun as Record<string, number>
    for (const key of Object.keys(fun)) {
      if (!(GROUP_GUARD_FUN_SETTING_KEYS as readonly string[]).includes(key)) {
        throw new Error(`group guard fun settings contains unsupported field: ${key}`)
      }
      const value = fun[key]
      parsedFunRecord[key] = readPositiveInteger(value, key, maxValueForFunSetting(key))
    }
    result.fun = parsedFun
  }
  if ('moderation' in record) {
    const moderation = requireRecord(record.moderation, 'group guard moderation settings')
    result.moderation = parseModerationSettings(moderation)
  }

  return result
}

function parseModerationSettings(input: PlainRecord): NonNullable<GroupGuardBehaviorSettingsInput['moderation']> {
  const parsed: NonNullable<GroupGuardBehaviorSettingsInput['moderation']> = {}
  const parsedRecord = parsed as Record<string, number | string | boolean>
  for (const key of Object.keys(input)) {
    if (!(GROUP_GUARD_MODERATION_SETTING_KEYS as readonly string[]).includes(key)) {
      throw new Error(`group guard moderation settings contains unsupported field: ${key}`)
    }
    const value = input[key]
    if (key === 'warningThresholdExpression') {
      parsedRecord[key] = readNonEmptyString(value, key, MAX_WARNING_EXPRESSION_LENGTH)
      continue
    }
    if (key === 'antiRecallNotify') {
      parsedRecord[key] = readBoolean(value, key)
      continue
    }
    parsedRecord[key] = readPositiveInteger(value, key, maxValueForModerationSetting(key))
  }
  return parsed
}

function readPositiveInteger(value: unknown, label: string, max: number): number {
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new Error(`${label} must be a positive integer`)
  }
  if (value > max) {
    throw new Error(`${label} must be at most ${max}`)
  }
  return value
}

function maxValueForFunSetting(key: string) {
  if (key === 'diceSides') return MAX_DICE_SIDES
  if (key === 'muteLotteryPityThreshold') return MAX_PITY_THRESHOLD
  return MAX_MUTE_SECONDS
}

function maxValueForModerationSetting(key: string) {
  if (key === 'defaultMuteSeconds') return MAX_MUTE_SECONDS
  return MAX_REPEAT_LIMIT
}

function readNonEmptyString(value: unknown, label: string, max: number) {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${label} must be a non-empty string`)
  }
  const trimmed = value.trim()
  if (trimmed.length > max) {
    throw new Error(`${label} must be at most ${max} characters`)
  }
  return trimmed
}

function readBoolean(value: unknown, label: string) {
  if (typeof value !== 'boolean') {
    throw new Error(`${label} must be a boolean`)
  }
  return value
}

function requireRecord(input: unknown, label: string): PlainRecord {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as PlainRecord
}
