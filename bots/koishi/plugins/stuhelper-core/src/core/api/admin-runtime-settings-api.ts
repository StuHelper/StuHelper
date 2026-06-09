import {
  ADMIN_MESSAGE_KEYS,
  AdminRuntimeSettingsStore,
  DEFAULT_ADMIN_RUNTIME_SETTINGS,
  syncAdminCommandDescriptions,
  type AdminRuntimeSettingsInput,
} from '@stuhelper/koishi-shared'

import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

const MAX_MESSAGE_LENGTH = 2000

type PlainRecord = Record<string, unknown>

export function registerAdminRuntimeSettingsAPI(api: WebSocketAPIContext): void {
  const store = new AdminRuntimeSettingsStore(api.ctx, DEFAULT_ADMIN_RUNTIME_SETTINGS)

  api.addAuthorityListener('stuhelperGroupCenter/admin-settings/get', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'admin settings')
      return success(await store.getSettings())
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '获取管理员命令设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/admin-settings/update', async function (params?: { settings?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'admin settings')
      const settings = parseAdminRuntimeSettingsInput(params?.settings)
      const saved = await store.saveSettings(settings)
      syncAdminCommandDescriptions(api.ctx, saved.messages)
      return success(saved)
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '保存管理员命令设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/admin-settings/reset', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'admin settings')
      const saved = await store.resetSettings()
      syncAdminCommandDescriptions(api.ctx, saved.messages)
      return success(saved)
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '重置管理员命令设置失败')
    }
  })
}

export function parseAdminRuntimeSettingsInput(input: unknown): AdminRuntimeSettingsInput {
  const record = requireRecord(input, 'admin settings')
  const result: AdminRuntimeSettingsInput = {}

  for (const key of Object.keys(record)) {
    if (key !== 'messages') {
      throw new Error(`admin settings contains unsupported field: ${key}`)
    }
  }

  if ('messages' in record) {
    const messages = requireRecord(record.messages, 'admin messages')
    const parsedMessages: NonNullable<AdminRuntimeSettingsInput['messages']> = {}
    const parsedMessageRecord = parsedMessages as Record<string, string>
    for (const key of Object.keys(messages)) {
      if (!(ADMIN_MESSAGE_KEYS as readonly string[]).includes(key)) {
        throw new Error(`admin messages contains unsupported field: ${key}`)
      }
      const value = messages[key]
      if (typeof value !== 'string') {
        throw new Error(`admin message ${key} must be a string`)
      }
      if (value.length > MAX_MESSAGE_LENGTH) {
        throw new Error(`admin message ${key} must be at most ${MAX_MESSAGE_LENGTH} characters`)
      }
      parsedMessageRecord[key] = value
    }
    result.messages = parsedMessages
  }

  return result
}

function requireRecord(input: unknown, label: string): PlainRecord {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as PlainRecord
}
