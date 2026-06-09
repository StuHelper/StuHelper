import {
  DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS,
  GROUP_GUARD_MESSAGE_KEYS,
  GroupGuardMessageSettingsStore,
  syncGroupGuardCommandDescriptions,
  type GroupGuardMessageSettingsInput,
} from '@stuhelper/koishi-shared'

import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

const MAX_MESSAGE_LENGTH = 3000

type PlainRecord = Record<string, unknown>

export function registerGroupGuardMessageSettingsAPI(api: WebSocketAPIContext): void {
  const store = new GroupGuardMessageSettingsStore(api.ctx, DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS)

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-message-settings/get', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard message settings')
      return success(await store.getSettings())
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '获取群管提示文案失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-message-settings/update', async function (params?: { settings?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard message settings')
      const settings = parseGroupGuardMessageSettingsInput(params?.settings)
      const saved = await store.saveSettings(settings)
      syncGroupGuardCommandDescriptions(api.ctx, saved.messages)
      return success(saved)
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '保存群管提示文案失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/group-guard-message-settings/reset', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'group guard message settings')
      const saved = await store.resetSettings()
      syncGroupGuardCommandDescriptions(api.ctx, saved.messages)
      return success(saved)
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '重置群管提示文案失败')
    }
  })
}

export function parseGroupGuardMessageSettingsInput(input: unknown): GroupGuardMessageSettingsInput {
  const record = requireRecord(input, 'group guard message settings')
  const result: GroupGuardMessageSettingsInput = {}

  for (const key of Object.keys(record)) {
    if (key !== 'messages') {
      throw new Error(`group guard message settings contains unsupported field: ${key}`)
    }
  }

  if ('messages' in record) {
    const messages = requireRecord(record.messages, 'group guard messages')
    const parsedMessages: NonNullable<GroupGuardMessageSettingsInput['messages']> = {}
    const parsedMessageRecord = parsedMessages as Record<string, string>
    for (const key of Object.keys(messages)) {
      if (!(GROUP_GUARD_MESSAGE_KEYS as readonly string[]).includes(key)) {
        throw new Error(`group guard messages contains unsupported field: ${key}`)
      }
      const value = messages[key]
      if (typeof value !== 'string') {
        throw new Error(`group guard message ${key} must be a string`)
      }
      if (value.length > MAX_MESSAGE_LENGTH) {
        throw new Error(`group guard message ${key} must be at most ${MAX_MESSAGE_LENGTH} characters`)
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
