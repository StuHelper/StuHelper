import type { Context } from 'koishi'

import { type SettingsReadCache, settingsReadCacheFor } from '../settings-cache'

export const GROUP_GUARD_AI_SETTINGS_TABLE = 'stuhelper_group_guard_ai_settings'

const DEFAULT_SETTINGS_ID = 'default'

declare module 'koishi' {
  interface Tables {
    [GROUP_GUARD_AI_SETTINGS_TABLE]: GroupGuardAISettingsRecord
  }
}

export interface GroupGuardAISettings {
  enabled: boolean
  endpoint: string
  apiKey: string
  model: string
}

export interface GroupGuardAISettingsRecord extends GroupGuardAISettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export interface GroupGuardAISettingsInput {
  enabled?: boolean
  endpoint?: string
  apiKey?: string
  model?: string
}

export const DEFAULT_GROUP_GUARD_AI_SETTINGS: GroupGuardAISettings = Object.freeze({
  enabled: false,
  endpoint: '',
  apiKey: '',
  model: '',
})

export function registerGroupGuardAISettingsModel(ctx: Context) {
  ctx.model.extend(GROUP_GUARD_AI_SETTINGS_TABLE, {
    id: 'string',
    enabled: { type: 'boolean', initial: false },
    endpoint: 'string',
    apiKey: 'string',
    model: 'string',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class GroupGuardAISettingsStore {
  private readonly cache: SettingsReadCache<GroupGuardAISettingsRecord>

  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: GroupGuardAISettings = DEFAULT_GROUP_GUARD_AI_SETTINGS,
  ) {
    this.cache = settingsReadCacheFor(ctx.database, GROUP_GUARD_AI_SETTINGS_TABLE)
  }

  async getSettings(): Promise<GroupGuardAISettingsRecord> {
    return this.cache.read(() => this.loadSettings())
  }

  private async loadSettings(): Promise<GroupGuardAISettingsRecord> {
    const [record] = await this.ctx.database.get(GROUP_GUARD_AI_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
    if (record) {
      return normalizeRecord(record, this.defaults)
    }
    return this.createDefaultRecord()
  }

  async saveSettings(input: GroupGuardAISettingsInput) {
    const current = await this.getSettings()
    const changes = normalizeInput(input)
    const now = new Date()
    const next: GroupGuardAISettingsRecord = {
      ...current,
      ...changes,
      updatedAt: now,
    }
    await this.ctx.database.set(GROUP_GUARD_AI_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return this.cache.write(next)
  }

  async resetSettings() {
    const current = await this.getSettings()
    const now = new Date()
    const next: GroupGuardAISettingsRecord = {
      ...current,
      ...normalizeAISettings(this.defaults, DEFAULT_GROUP_GUARD_AI_SETTINGS),
      updatedAt: now,
    }
    await this.ctx.database.set(GROUP_GUARD_AI_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      enabled: next.enabled,
      endpoint: next.endpoint,
      apiKey: next.apiKey,
      model: next.model,
      updatedAt: now,
    })
    return this.cache.write(next)
  }

  async getAISettings() {
    const settings = await this.getSettings()
    return {
      enabled: settings.enabled,
      endpoint: settings.endpoint,
      apiKey: settings.apiKey,
      model: settings.model,
    }
  }

  private async createDefaultRecord() {
    const now = new Date()
    const settings = normalizeAISettings(this.defaults, DEFAULT_GROUP_GUARD_AI_SETTINGS)
    const record: GroupGuardAISettingsRecord = {
      id: DEFAULT_SETTINGS_ID,
      ...settings,
      createdAt: now,
      updatedAt: now,
    }
    try {
      await this.ctx.database.create(GROUP_GUARD_AI_SETTINGS_TABLE, record)
    } catch (error) {
      const [existing] = await this.ctx.database.get(GROUP_GUARD_AI_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
      if (existing) {
        return normalizeRecord(existing, this.defaults)
      }
      throw error
    }
    return record
  }
}

function normalizeRecord(
  record: GroupGuardAISettingsRecord,
  defaults: GroupGuardAISettings,
): GroupGuardAISettingsRecord {
  return {
    ...record,
    ...normalizeAISettings(record, defaults),
  }
}

function normalizeInput(input: GroupGuardAISettingsInput): Partial<GroupGuardAISettings> {
  const changes: Partial<GroupGuardAISettings> = {}
  if (typeof input.enabled === 'boolean') {
    changes.enabled = input.enabled
  }
  if (typeof input.endpoint === 'string') {
    changes.endpoint = input.endpoint.trim()
  }
  if (typeof input.apiKey === 'string') {
    changes.apiKey = input.apiKey.trim()
  }
  if (typeof input.model === 'string') {
    changes.model = input.model.trim()
  }
  return changes
}

function normalizeAISettings(value: unknown, fallback: GroupGuardAISettings): GroupGuardAISettings {
  const record = isObject(value) ? value : {}
  return {
    enabled: typeof record.enabled === 'boolean' ? record.enabled : fallback.enabled,
    endpoint: stringOrDefault(record.endpoint, fallback.endpoint),
    apiKey: stringOrDefault(record.apiKey, fallback.apiKey),
    model: stringOrDefault(record.model, fallback.model),
  }
}

function stringOrDefault(value: unknown, fallback: string) {
  return typeof value === 'string' ? value.trim() : fallback
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
