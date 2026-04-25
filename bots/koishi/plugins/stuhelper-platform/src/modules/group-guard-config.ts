import type {
  StuhelperAIConfig,
  StuhelperGroupGuardPluginConfig,
  StuhelperKeywordRuleConfig,
} from '@stuhelper/koishi-shared'

import type { StuhelperModuleConfig } from '../module-contract'
import {
  readBoolean,
  readEnum,
  readNonNegativeInteger,
  readOptionalRecord,
  readPositiveInteger,
  readRecord,
  readRequiredString,
  readString,
  readStringArray,
  readText,
  rejectUnknownFields,
} from './group-guard-readers'

const DEFAULT_MUTE_DURATION_SECONDS = 600
const DEFAULT_KICK_AFTER_MINUTES = 30
const DEFAULT_SCAN_INTERVAL_SECONDS = 60
const DEFAULT_REPEAT_THRESHOLD = 3
const DEFAULT_REPEAT_WINDOW_SIZE = 3
const DEFAULT_WARNING_THRESHOLD_EXPRESSION = `warnings >= ${DEFAULT_REPEAT_THRESHOLD}`
const DEFAULT_MUTE_SECONDS = 180
const DEFAULT_DICE_SIDES = 100
const DEFAULT_MUTE_BASE_SECONDS = 120
const DEFAULT_MUTE_MAX_SECONDS = 600
const DEFAULT_PITY_THRESHOLD = 5
const DEFAULT_PITY_SECONDS = 300
const MIN_DICE_SIDES = 2

export type GroupGuardConfig = StuhelperGroupGuardPluginConfig & StuhelperModuleConfig

const DEFAULT_CONFIG: GroupGuardConfig = {
  platform: { baseUrl: 'http://127.0.0.1:8080', serviceToken: '' },
  guard: {
    targetGroups: [],
    muteDurationSeconds: DEFAULT_MUTE_DURATION_SECONDS,
    kickAfterMinutes: DEFAULT_KICK_AFTER_MINUTES,
    reminderTemplate: '请先完成 StuHelper 注册、QQ 绑定与学生认证。',
    exemptUsers: [],
  },
  scheduler: { scanIntervalSeconds: DEFAULT_SCAN_INTERVAL_SECONDS },
  moderation: {
    repeatThreshold: DEFAULT_REPEAT_THRESHOLD,
    repeatWindowSize: DEFAULT_REPEAT_WINDOW_SIZE,
    warningThresholdExpression: DEFAULT_WARNING_THRESHOLD_EXPRESSION,
    defaultMuteSeconds: DEFAULT_MUTE_SECONDS,
    antiRecallNotify: true,
    keywordRules: [],
  },
  fun: {
    diceSides: DEFAULT_DICE_SIDES,
    muteLotteryBaseSeconds: DEFAULT_MUTE_BASE_SECONDS,
    muteLotteryMaxSeconds: DEFAULT_MUTE_MAX_SECONDS,
    muteLotteryPityThreshold: DEFAULT_PITY_THRESHOLD,
    muteLotteryPitySeconds: DEFAULT_PITY_SECONDS,
  },
  ai: { enabled: false, endpoint: '', apiKey: '', model: '' },
}

export function createDefaultGroupGuardConfig(): GroupGuardConfig {
  return structuredClone(DEFAULT_CONFIG)
}

export function parseGroupGuardConfig(value: unknown): GroupGuardConfig {
  const record = readRecord(value, 'group guard config')
  rejectUnknownFields(record, ['platform', 'guard', 'scheduler', 'moderation', 'fun', 'ai'])

  return {
    platform: parsePlatform(readOptionalRecord(record.platform, DEFAULT_CONFIG.platform)),
    guard: parseGuard(readOptionalRecord(record.guard, DEFAULT_CONFIG.guard)),
    scheduler: parseScheduler(readOptionalRecord(record.scheduler, DEFAULT_CONFIG.scheduler)),
    moderation: parseModeration(readOptionalRecord(record.moderation, DEFAULT_CONFIG.moderation)),
    fun: parseFun(readOptionalRecord(record.fun, DEFAULT_CONFIG.fun)),
    ai: parseAI(readOptionalRecord(record.ai, DEFAULT_CONFIG.ai)),
  }
}

export function toGroupGuardPluginConfig(config: GroupGuardConfig): StuhelperGroupGuardPluginConfig {
  return structuredClone(config)
}

function parsePlatform(record: Record<string, unknown>) {
  rejectUnknownFields(record, ['baseUrl', 'serviceToken'])
  return {
    baseUrl: readString(record.baseUrl, 'platform.baseUrl', DEFAULT_CONFIG.platform.baseUrl),
    serviceToken: readText(
      record.serviceToken,
      'platform.serviceToken',
      DEFAULT_CONFIG.platform.serviceToken,
    ),
  }
}

function parseGuard(record: Record<string, unknown>) {
  rejectUnknownFields(record, [
    'targetGroups',
    'muteDurationSeconds',
    'kickAfterMinutes',
    'reminderTemplate',
    'exemptUsers',
  ])
  return {
    targetGroups: readStringArray(record.targetGroups, 'guard.targetGroups'),
    muteDurationSeconds: readPositiveInteger({
      value: record.muteDurationSeconds,
      field: 'guard.muteDurationSeconds',
      defaultValue: DEFAULT_CONFIG.guard.muteDurationSeconds,
    }),
    kickAfterMinutes: readPositiveInteger({
      value: record.kickAfterMinutes,
      field: 'guard.kickAfterMinutes',
      defaultValue: DEFAULT_CONFIG.guard.kickAfterMinutes,
    }),
    reminderTemplate: readString(record.reminderTemplate, 'guard.reminderTemplate', DEFAULT_CONFIG.guard.reminderTemplate),
    exemptUsers: readStringArray(record.exemptUsers, 'guard.exemptUsers'),
  }
}

function parseScheduler(record: Record<string, unknown>) {
  rejectUnknownFields(record, ['scanIntervalSeconds'])
  return {
    scanIntervalSeconds: readPositiveInteger({
      value: record.scanIntervalSeconds,
      field: 'scheduler.scanIntervalSeconds',
      defaultValue: DEFAULT_CONFIG.scheduler.scanIntervalSeconds,
    }),
  }
}

function parseModeration(record: Record<string, unknown>) {
  rejectUnknownFields(record, [
    'repeatThreshold',
    'repeatWindowSize',
    'warningThresholdExpression',
    'defaultMuteSeconds',
    'antiRecallNotify',
    'keywordRules',
  ])
  return {
    repeatThreshold: readPositiveInteger({
      value: record.repeatThreshold,
      field: 'moderation.repeatThreshold',
      defaultValue: DEFAULT_CONFIG.moderation.repeatThreshold,
    }),
    repeatWindowSize: readPositiveInteger({
      value: record.repeatWindowSize,
      field: 'moderation.repeatWindowSize',
      defaultValue: DEFAULT_CONFIG.moderation.repeatWindowSize,
    }),
    warningThresholdExpression: readString(record.warningThresholdExpression, 'moderation.warningThresholdExpression', DEFAULT_CONFIG.moderation.warningThresholdExpression),
    defaultMuteSeconds: readPositiveInteger({
      value: record.defaultMuteSeconds,
      field: 'moderation.defaultMuteSeconds',
      defaultValue: DEFAULT_CONFIG.moderation.defaultMuteSeconds,
    }),
    antiRecallNotify: readBoolean(record.antiRecallNotify, 'moderation.antiRecallNotify', DEFAULT_CONFIG.moderation.antiRecallNotify),
    keywordRules: readKeywordRules(record.keywordRules),
  }
}

function parseFun(record: Record<string, unknown>) {
  rejectUnknownFields(record, [
    'diceSides',
    'muteLotteryBaseSeconds',
    'muteLotteryMaxSeconds',
    'muteLotteryPityThreshold',
    'muteLotteryPitySeconds',
  ])
  return {
    diceSides: readPositiveInteger({
      value: record.diceSides,
      field: 'fun.diceSides',
      defaultValue: DEFAULT_CONFIG.fun.diceSides,
      minimum: MIN_DICE_SIDES,
    }),
    muteLotteryBaseSeconds: readPositiveInteger({
      value: record.muteLotteryBaseSeconds,
      field: 'fun.muteLotteryBaseSeconds',
      defaultValue: DEFAULT_CONFIG.fun.muteLotteryBaseSeconds,
    }),
    muteLotteryMaxSeconds: readPositiveInteger({
      value: record.muteLotteryMaxSeconds,
      field: 'fun.muteLotteryMaxSeconds',
      defaultValue: DEFAULT_CONFIG.fun.muteLotteryMaxSeconds,
    }),
    muteLotteryPityThreshold: readPositiveInteger({
      value: record.muteLotteryPityThreshold,
      field: 'fun.muteLotteryPityThreshold',
      defaultValue: DEFAULT_CONFIG.fun.muteLotteryPityThreshold,
    }),
    muteLotteryPitySeconds: readPositiveInteger({
      value: record.muteLotteryPitySeconds,
      field: 'fun.muteLotteryPitySeconds',
      defaultValue: DEFAULT_CONFIG.fun.muteLotteryPitySeconds,
    }),
  }
}

function parseAI(record: Record<string, unknown>): StuhelperAIConfig {
  rejectUnknownFields(record, ['enabled', 'endpoint', 'apiKey', 'model'])
  return {
    enabled: readBoolean(record.enabled, 'ai.enabled', DEFAULT_CONFIG.ai.enabled),
    endpoint: readText(record.endpoint, 'ai.endpoint', DEFAULT_CONFIG.ai.endpoint),
    apiKey: readText(record.apiKey, 'ai.apiKey', DEFAULT_CONFIG.ai.apiKey),
    model: readText(record.model, 'ai.model', DEFAULT_CONFIG.ai.model),
  }
}

function readKeywordRules(value: unknown): StuhelperKeywordRuleConfig[] {
  if (value === undefined) return []
  if (!Array.isArray(value)) throw new Error('moderation.keywordRules must be an array')
  return value.map((item, index) => parseKeywordRule(item, index))
}

function parseKeywordRule(value: unknown, index: number): StuhelperKeywordRuleConfig {
  const record = readRecord(value, `moderation.keywordRules[${index}]`)
  rejectUnknownFields(record, ['id', 'guildId', 'pattern', 'matchMode', 'action', 'enabled', 'muteSeconds', 'note'])
  return {
    id: readRequiredString(record.id, `moderation.keywordRules[${index}].id`),
    guildId: readString(record.guildId, `moderation.keywordRules[${index}].guildId`, '*'),
    pattern: readRequiredString(record.pattern, `moderation.keywordRules[${index}].pattern`),
    matchMode: readEnum({
      value: record.matchMode,
      field: `moderation.keywordRules[${index}].matchMode`,
      allowed: ['includes', 'regex'],
      defaultValue: 'includes',
    }),
    action: readEnum({
      value: record.action,
      field: `moderation.keywordRules[${index}].action`,
      allowed: ['warn', 'delete', 'mute', 'review'],
      defaultValue: 'warn',
    }),
    enabled: readBoolean(record.enabled, `moderation.keywordRules[${index}].enabled`, true),
    muteSeconds: readNonNegativeInteger(record.muteSeconds, `moderation.keywordRules[${index}].muteSeconds`, 0),
    note: readText(record.note, `moderation.keywordRules[${index}].note`, ''),
  }
}
