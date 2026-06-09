import {
  BINDING_MESSAGE_KEYS,
  BindingRuntimeSettingsStore,
  DEFAULT_BINDING_RUNTIME_SETTINGS,
  type BindingRuntimeSettingsInput,
} from '@stuhelper/koishi-shared'

import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

const MAX_COMMAND_LENGTH = 32
const MAX_MESSAGE_LENGTH = 2000

type PlainRecord = Record<string, unknown>

export function registerBindingRuntimeSettingsAPI(api: WebSocketAPIContext): void {
  const store = new BindingRuntimeSettingsStore(api.ctx, DEFAULT_BINDING_RUNTIME_SETTINGS)

  api.addAuthorityListener('stuhelperGroupCenter/binding-settings/get', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'binding settings')
      return success(await store.getSettings())
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '获取绑定设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/binding-settings/update', async function (params?: { settings?: unknown }) {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'binding settings')
      const settings = parseBindingRuntimeSettingsInput(params?.settings)
      const saved = await store.saveSettings(settings)
      return success(saved)
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '保存绑定设置失败')
    }
  })

  api.addAuthorityListener('stuhelperGroupCenter/binding-settings/reset', async function () {
    try {
      const scope = await api.resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'binding settings')
      const saved = await store.resetSettings()
      return success(saved)
    } catch (cause) {
      return error(cause instanceof Error ? cause.message : '重置绑定设置失败')
    }
  })
}

export function parseBindingRuntimeSettingsInput(input: unknown): BindingRuntimeSettingsInput {
  const record = requireRecord(input, 'binding settings')
  const result: BindingRuntimeSettingsInput = {}

  for (const key of Object.keys(record)) {
    if (key !== 'command' && key !== 'messages') {
      throw new Error(`binding settings contains unsupported field: ${key}`)
    }
  }

  if ('command' in record) {
    if (typeof record.command !== 'string') {
      throw new Error('binding command must be a string')
    }
    const command = record.command.trim()
    if (!command) {
      throw new Error('binding command is required')
    }
    if (command.length > MAX_COMMAND_LENGTH) {
      throw new Error(`binding command must be at most ${MAX_COMMAND_LENGTH} characters`)
    }
    result.command = command
  }

  if ('messages' in record) {
    const messages = requireRecord(record.messages, 'binding messages')
    const parsedMessages: NonNullable<BindingRuntimeSettingsInput['messages']> = {}
    const parsedMessageRecord = parsedMessages as Record<string, string>
    for (const key of Object.keys(messages)) {
      if (!(BINDING_MESSAGE_KEYS as readonly string[]).includes(key)) {
        throw new Error(`binding messages contains unsupported field: ${key}`)
      }
      const value = messages[key]
      if (typeof value !== 'string') {
        throw new Error(`binding message ${key} must be a string`)
      }
      if (value.length > MAX_MESSAGE_LENGTH) {
        throw new Error(`binding message ${key} must be at most ${MAX_MESSAGE_LENGTH} characters`)
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
