import {
  DEFAULT_GROUP_GUARD_AI_SETTINGS,
  GroupGuardAISettingsStore,
  type GroupGuardAISettingsInput,
  type GroupGuardAISettingsRecord,
} from '@stuhelper/koishi-shared'

import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

type PlainRecord = Record<string, unknown>

const MAX_ENDPOINT_LENGTH = 2048
const MAX_API_KEY_LENGTH = 4096
const MAX_MODEL_LENGTH = 256

export type GroupGuardAIPublicSettings = Omit<GroupGuardAISettingsRecord, 'apiKey'> & {
  apiKeyConfigured: boolean
  apiKeyMasked: string
}

export interface GroupGuardAISettingsUpdateInput {
  enabled?: boolean
  endpoint?: string
  model?: string
  newApiKey?: string
  clearApiKey?: boolean
}

export function registerGroupGuardAISettingsAPI(api: WebSocketAPIContext): void {
  const store = new GroupGuardAISettingsStore(api.ctx, DEFAULT_GROUP_GUARD_AI_SETTINGS)

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-ai-settings/get', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard ai settings')
      return success(toPublicAISettings(await store.getSettings()))
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '获取群管 AI 设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-ai-settings/update', async function (params?: { settings?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard ai settings')
      const settings = parseGroupGuardAISettingsInput(params?.settings)
      return success(toPublicAISettings(await store.saveSettings(settings)))
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '保存群管 AI 设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-ai-settings/reset', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard ai settings')
      return success(toPublicAISettings(await store.resetSettings()))
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '重置群管 AI 设置失败')
    }
  })
}

export function parseGroupGuardAISettingsInput(input: unknown): GroupGuardAISettingsInput {
  const record = requireRecord(input, 'group guard ai settings')
  const result: GroupGuardAISettingsInput = {}

  for (const key of Object.keys(record)) {
    if (!GROUP_GUARD_AI_UPDATE_KEYS.includes(key)) {
      throw new Error(`group guard ai settings contains unsupported field: ${key}`)
    }
  }

  if ('enabled' in record) {
    result.enabled = readBoolean(record.enabled, 'enabled')
  }
  if ('endpoint' in record) {
    result.endpoint = readString(record.endpoint, 'endpoint', MAX_ENDPOINT_LENGTH)
  }
  if ('model' in record) {
    result.model = readString(record.model, 'model', MAX_MODEL_LENGTH)
  }

  const clearApiKey = record.clearApiKey === true
  const newApiKey = typeof record.newApiKey === 'string' ? record.newApiKey.trim() : ''
  if ('clearApiKey' in record && typeof record.clearApiKey !== 'boolean') {
    throw new Error('clearApiKey must be a boolean')
  }
  if ('newApiKey' in record) {
    readString(record.newApiKey, 'newApiKey', MAX_API_KEY_LENGTH)
  }
  if (clearApiKey && newApiKey) {
    throw new Error('不能同时清除和替换 API Key')
  }
  if (clearApiKey) {
    result.apiKey = ''
  } else if (newApiKey) {
    result.apiKey = newApiKey
  }

  return result
}

export function toPublicAISettings(settings: GroupGuardAISettingsRecord): GroupGuardAIPublicSettings {
  const { apiKey, ...rest } = settings
  return {
    ...rest,
    apiKeyConfigured: apiKey.length > 0,
    apiKeyMasked: maskSecret(apiKey),
  }
}

const GROUP_GUARD_AI_UPDATE_KEYS = Object.freeze([
  'enabled',
  'endpoint',
  'model',
  'newApiKey',
  'clearApiKey',
])

function readBoolean(value: unknown, label: string) {
  if (typeof value !== 'boolean') {
    throw new Error(`${label} must be a boolean`)
  }
  return value
}

function readString(value: unknown, label: string, max: number) {
  if (typeof value !== 'string') {
    throw new Error(`${label} must be a string`)
  }
  const trimmed = value.trim()
  if (trimmed.length > max) {
    throw new Error(`${label} must be at most ${max} characters`)
  }
  return trimmed
}

function maskSecret(value: string): string {
  if (!value) return ''
  if (value.length <= 8) return '已配置'
  return `${value.slice(0, 4)}...${value.slice(-4)}`
}

function requireRecord(input: unknown, label: string): PlainRecord {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as PlainRecord
}
