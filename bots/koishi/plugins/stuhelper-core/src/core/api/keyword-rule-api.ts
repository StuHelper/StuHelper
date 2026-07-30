import {
  ModerationStore,
  type KeywordActionType,
  type KeywordMatchMode,
  type KeywordRuleRecord,
} from '@stuhelper/koishi-moderation-core'
import { assertSafeKeywordRegex } from '@stuhelper/koishi-shared'

import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  type ConsoleGuildScope,
} from './console-guild-scope'

type PlainRecord = Record<string, unknown>

const GLOBAL_GUILD_ID = '*'
const MAX_ID_LENGTH = 128
const MAX_GUILD_ID_LENGTH = 64
const MAX_PATTERN_LENGTH = 256
const MAX_NOTE_LENGTH = 512
const MAX_MUTE_SECONDS = 2_592_000
const ID_PATTERN = /^[A-Za-z0-9._:-]+$/
const MATCH_MODES = new Set<KeywordMatchMode>(['includes', 'regex'])
const ACTION_TYPES = new Set<KeywordActionType>(['warn', 'delete', 'mute', 'review'])

export interface KeywordRuleInput {
  id: string
  guildId: string
  pattern: string
  matchMode: KeywordMatchMode
  action: KeywordActionType
  enabled: boolean
  muteSeconds: number
  note: string | null
}

export interface KeywordRulePublicRecord extends KeywordRuleInput {
  createdAt: string
  updatedAt: string
}

export function registerKeywordRuleAPI(api: WebSocketAPIContext): void {
  const store = new ModerationStore(api.ctx)

  api.addAuthorityListener('stuhelperGroupCenter/keyword-rules/list', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      const records = await store.listAllKeywordRules()
      return success(records
        .filter((record) => canReadRule(scope, record))
        .sort(sortKeywordRules)
        .map(toPublicKeywordRule))
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '获取关键词规则失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/keyword-rules/upsert', async function (params?: { rule?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      const rule = parseKeywordRuleInput(params?.rule)
      assertKeywordRuleScope(scope, rule.guildId, 'keyword rule')
      const existing = await store.getKeywordRule(rule.id)
      if (existing) {
        assertKeywordRuleScope(scope, existing.guildId, 'keyword rule')
      }
      const now = new Date()
      const record: KeywordRuleRecord = {
        ...rule,
        createdAt: existing?.createdAt ?? now,
        updatedAt: now,
      }
      await store.upsertKeywordRule(record)
      return success(toPublicKeywordRule(record))
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '保存关键词规则失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/keyword-rules/delete', async function (params?: { id?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      const id = readRuleID(params?.id)
      await deleteKeywordRuleIfPresent(store, scope, id)
      return success({ success: true })
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '删除关键词规则失败')
    }
  })
}

export interface KeywordRuleDeleteStore {
  getKeywordRule: (id: string) => Promise<KeywordRuleRecord | undefined>
  deleteKeywordRule: (id: string) => Promise<unknown>
}

export async function deleteKeywordRuleIfPresent(
  store: KeywordRuleDeleteStore,
  scope: ConsoleGuildScope,
  id: string,
): Promise<boolean> {
  const existing = await store.getKeywordRule(id)
  if (!existing) {
    return false
  }
  assertKeywordRuleScope(scope, existing.guildId, 'keyword rule')
  await store.deleteKeywordRule(id)
  return true
}

export function parseKeywordRuleInput(input: unknown): KeywordRuleInput {
  const record = requireRecord(input, 'keyword rule')
  for (const key of Object.keys(record)) {
    if (!KEYWORD_RULE_INPUT_KEYS.includes(key)) {
      throw new Error(`keyword rule contains unsupported field: ${key}`)
    }
  }

  const id = readRuleID(record.id)
  const guildId = readGuildID(record.guildId)
  const pattern = readPattern(record.pattern)
  const matchMode = readMatchMode(record.matchMode)
  const action = readAction(record.action)
  const enabled = readBoolean(record.enabled, 'enabled')
  const muteSeconds = readMuteSeconds(record.muteSeconds)
  const note = readNullableNote(record.note)

  if (matchMode === 'regex') {
    assertSafeKeywordRegex(pattern, MAX_PATTERN_LENGTH)
  }

  return {
    id,
    guildId,
    pattern,
    matchMode,
    action,
    enabled,
    muteSeconds,
    note,
  }
}

export function toPublicKeywordRule(record: KeywordRuleRecord): KeywordRulePublicRecord {
  return {
    id: record.id,
    guildId: record.guildId,
    pattern: record.pattern,
    matchMode: record.matchMode,
    action: record.action,
    enabled: record.enabled,
    muteSeconds: record.muteSeconds,
    note: record.note,
    createdAt: record.createdAt.toISOString(),
    updatedAt: record.updatedAt.toISOString(),
  }
}

function assertKeywordRuleScope(scope: ConsoleGuildScope, guildId: string, resource: string) {
  if (guildId === GLOBAL_GUILD_ID) {
    assertGlobalConsoleScope(scope, resource)
    return
  }
  assertConsoleGuildAccess(scope, guildId, resource)
}

function canReadRule(scope: ConsoleGuildScope, record: KeywordRuleRecord) {
  if (record.guildId === GLOBAL_GUILD_ID) {
    return scope.kind === 'all'
  }
  if (scope.kind === 'all') {
    return true
  }
  return scope.guildIds.has(record.guildId)
}

const KEYWORD_RULE_INPUT_KEYS = Object.freeze([
  'id',
  'guildId',
  'pattern',
  'matchMode',
  'action',
  'enabled',
  'muteSeconds',
  'note',
])

function readRuleID(value: unknown) {
  const id = readString(value, 'id', MAX_ID_LENGTH)
  if (!ID_PATTERN.test(id)) {
    throw new Error('id must only contain letters, numbers, dot, underscore, colon or hyphen')
  }
  return id
}

function readGuildID(value: unknown) {
  const guildId = readString(value, 'guildId', MAX_GUILD_ID_LENGTH)
  if (guildId === GLOBAL_GUILD_ID) {
    return guildId
  }
  if (!/^\d+$/.test(guildId)) {
    throw new Error('guildId must be a numeric group id or *')
  }
  return guildId
}

function readPattern(value: unknown) {
  return readString(value, 'pattern', MAX_PATTERN_LENGTH)
}

function readMatchMode(value: unknown): KeywordMatchMode {
  if (typeof value !== 'string' || !MATCH_MODES.has(value as KeywordMatchMode)) {
    throw new Error('matchMode must be includes or regex')
  }
  return value as KeywordMatchMode
}

function readAction(value: unknown): KeywordActionType {
  if (typeof value !== 'string' || !ACTION_TYPES.has(value as KeywordActionType)) {
    throw new Error('action must be warn, delete, mute or review')
  }
  return value as KeywordActionType
}

function readMuteSeconds(value: unknown) {
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    throw new Error('muteSeconds must be a non-negative integer')
  }
  if (value > MAX_MUTE_SECONDS) {
    throw new Error(`muteSeconds must be at most ${MAX_MUTE_SECONDS}`)
  }
  return value
}

function readNullableNote(value: unknown) {
  if (value === null || value === undefined) {
    return null
  }
  const note = readString(value, 'note', MAX_NOTE_LENGTH, true)
  return note || null
}

function readBoolean(value: unknown, label: string) {
  if (typeof value !== 'boolean') {
    throw new Error(`${label} must be a boolean`)
  }
  return value
}

function readString(value: unknown, label: string, max: number, allowEmpty = false) {
  if (typeof value !== 'string') {
    throw new Error(`${label} must be a string`)
  }
  const trimmed = value.trim()
  if (!allowEmpty && !trimmed) {
    throw new Error(`${label} is required`)
  }
  if (trimmed.length > max) {
    throw new Error(`${label} must be at most ${max} characters`)
  }
  return trimmed
}

function requireRecord(input: unknown, label: string): PlainRecord {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as PlainRecord
}

function sortKeywordRules(left: KeywordRuleRecord, right: KeywordRuleRecord) {
  const scopeOrder = ruleScopeRank(left.guildId) - ruleScopeRank(right.guildId)
  if (scopeOrder !== 0) return scopeOrder
  const guildOrder = left.guildId.localeCompare(right.guildId)
  if (guildOrder !== 0) return guildOrder
  return left.id.localeCompare(right.id)
}

function ruleScopeRank(guildId: string) {
  return guildId === GLOBAL_GUILD_ID ? 0 : 1
}
