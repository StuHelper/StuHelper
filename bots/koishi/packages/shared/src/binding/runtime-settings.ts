import type { Context } from 'koishi'

import { type SettingsReadCache, settingsReadCacheFor } from '../settings-cache'
import {
  DEFAULT_BINDING_MESSAGES,
  resolveBindingMessages,
} from '../message-template'
import type { StuhelperBindingMessageConfig } from '../types/index'

export const BINDING_RUNTIME_SETTINGS_TABLE = 'stuhelper_binding_runtime_settings'
export const DEFAULT_BINDING_COMMAND = '绑定'

const DEFAULT_SETTINGS_ID = 'default'

export const BINDING_MESSAGE_KEYS = Object.freeze([
  'directOnly',
  'missingCode',
  'successVerified',
  'successUnverified',
  'unavailable',
  'invalidCode',
  'unauthorized',
  'conflict',
  'notConfigured',
  'rateLimited',
  'tooManyFailures',
] as const)

declare module 'koishi' {
  interface Tables {
    [BINDING_RUNTIME_SETTINGS_TABLE]: BindingRuntimeSettingsRecord
  }
}

export interface BindingRuntimeSettings {
  command: string
  messages: StuhelperBindingMessageConfig
}

export interface BindingRuntimeSettingsRecord extends BindingRuntimeSettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export interface BindingRuntimeSettingsInput {
  command?: string
  messages?: Partial<StuhelperBindingMessageConfig>
}

export const DEFAULT_BINDING_RUNTIME_SETTINGS: BindingRuntimeSettings = {
  command: DEFAULT_BINDING_COMMAND,
  messages: DEFAULT_BINDING_MESSAGES,
}

export function registerBindingRuntimeSettingsModel(ctx: Context) {
  ctx.model.extend(BINDING_RUNTIME_SETTINGS_TABLE, {
    id: 'string',
    command: 'string',
    messages: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class BindingRuntimeSettingsStore {
  private readonly cache: SettingsReadCache<BindingRuntimeSettingsRecord>

  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: BindingRuntimeSettings = DEFAULT_BINDING_RUNTIME_SETTINGS,
  ) {
    this.cache = settingsReadCacheFor(ctx.database, BINDING_RUNTIME_SETTINGS_TABLE)
  }

  async getSettings(): Promise<BindingRuntimeSettingsRecord> {
    return this.cache.read(() => this.loadSettings())
  }

  private async loadSettings(): Promise<BindingRuntimeSettingsRecord> {
    const [record] = await this.ctx.database.get(BINDING_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
    if (record) {
      return normalizeRecord(record, this.defaults)
    }
    return this.createDefaultRecord()
  }

  async saveSettings(input: BindingRuntimeSettingsInput) {
    const current = await this.getSettings()
    const changes = normalizeInput(input, current, this.defaults)
    const now = new Date()
    const next: BindingRuntimeSettingsRecord = {
      ...current,
      ...changes,
      updatedAt: now,
    }
    await this.ctx.database.set(BINDING_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return this.cache.write(next)
  }

  async resetSettings() {
    const current = await this.getSettings()
    const now = new Date()
    const next: BindingRuntimeSettingsRecord = {
      ...current,
      ...this.defaults,
      messages: resolveBindingMessages(this.defaults.messages),
      updatedAt: now,
    }
    await this.ctx.database.set(BINDING_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      command: next.command,
      messages: next.messages,
      updatedAt: now,
    })
    return this.cache.write(next)
  }

  async getCommand() {
    return (await this.getSettings()).command
  }

  async getMessages() {
    return (await this.getSettings()).messages
  }

  private async createDefaultRecord() {
    const now = new Date()
    const record: BindingRuntimeSettingsRecord = {
      id: DEFAULT_SETTINGS_ID,
      command: normalizeCommand(this.defaults.command, DEFAULT_BINDING_COMMAND),
      messages: resolveBindingMessages(this.defaults.messages),
      createdAt: now,
      updatedAt: now,
    }
    try {
      await this.ctx.database.create(BINDING_RUNTIME_SETTINGS_TABLE, record)
    } catch (error) {
      const [existing] = await this.ctx.database.get(BINDING_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
      if (existing) {
        return normalizeRecord(existing, this.defaults)
      }
      throw error
    }
    return record
  }
}

function normalizeRecord(
  record: BindingRuntimeSettingsRecord,
  defaults: BindingRuntimeSettings,
): BindingRuntimeSettingsRecord {
  return {
    ...record,
    command: normalizeCommand(record.command, defaults.command),
    messages: resolveBindingMessages(isObject(record.messages) ? record.messages : defaults.messages),
  }
}

function normalizeInput(
  input: BindingRuntimeSettingsInput,
  current: BindingRuntimeSettings,
  defaults: BindingRuntimeSettings,
): Partial<BindingRuntimeSettings> {
  const changes: Partial<BindingRuntimeSettings> = {}
  const command = normalizeOptionalCommand(input.command)
  if (command) {
    changes.command = command
  }
  const messages = normalizeMessages(input.messages)
  if (messages) {
    changes.messages = resolveBindingMessages({
      ...defaults.messages,
      ...current.messages,
      ...messages,
    })
  }
  return changes
}

function normalizeOptionalCommand(value: unknown) {
  if (typeof value !== 'string') {
    return undefined
  }
  const trimmed = value.trim()
  return trimmed || undefined
}

function normalizeCommand(value: unknown, fallback: string) {
  return normalizeOptionalCommand(value) || fallback
}

function normalizeMessages(value: unknown) {
  if (!isObject(value)) {
    return undefined
  }
  const result: Partial<StuhelperBindingMessageConfig> = {}
  for (const key of BINDING_MESSAGE_KEYS) {
    const message = value[key]
    if (typeof message === 'string') {
      result[key] = message
    }
  }
  return Object.keys(result).length ? result : undefined
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
