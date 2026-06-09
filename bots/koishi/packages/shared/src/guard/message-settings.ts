import type { Context } from 'koishi'

import {
  DEFAULT_GROUP_GUARD_MESSAGES,
  resolveGroupGuardMessages,
} from '../message-template'
import type { StuhelperGroupGuardMessageConfig } from '../types/index'

export const GROUP_GUARD_MESSAGE_SETTINGS_TABLE = 'stuhelper_group_guard_message_settings'

const DEFAULT_SETTINGS_ID = 'default'

export const GROUP_GUARD_MESSAGE_KEYS = Object.freeze(
  Object.keys(DEFAULT_GROUP_GUARD_MESSAGES) as Array<keyof StuhelperGroupGuardMessageConfig>,
)

declare module 'koishi' {
  interface Tables {
    [GROUP_GUARD_MESSAGE_SETTINGS_TABLE]: GroupGuardMessageSettingsRecord
  }
}

export interface GroupGuardMessageSettings {
  messages: StuhelperGroupGuardMessageConfig
}

export interface GroupGuardMessageSettingsRecord extends GroupGuardMessageSettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export interface GroupGuardMessageSettingsInput {
  messages?: Partial<StuhelperGroupGuardMessageConfig>
}

export const DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS: GroupGuardMessageSettings = {
  messages: DEFAULT_GROUP_GUARD_MESSAGES,
}

export function registerGroupGuardMessageSettingsModel(ctx: Context) {
  ctx.model.extend(GROUP_GUARD_MESSAGE_SETTINGS_TABLE, {
    id: 'string',
    messages: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class GroupGuardMessageSettingsStore {
  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: GroupGuardMessageSettings = DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS,
  ) {}

  async getSettings(): Promise<GroupGuardMessageSettingsRecord> {
    const [record] = await this.ctx.database.get(GROUP_GUARD_MESSAGE_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
    if (record) {
      return normalizeRecord(record, this.defaults)
    }
    return this.createDefaultRecord()
  }

  async saveSettings(input: GroupGuardMessageSettingsInput) {
    const current = await this.getSettings()
    const changes = normalizeInput(input, current, this.defaults)
    const now = new Date()
    const next: GroupGuardMessageSettingsRecord = {
      ...current,
      ...changes,
      updatedAt: now,
    }
    await this.ctx.database.set(GROUP_GUARD_MESSAGE_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return next
  }

  async resetSettings() {
    const current = await this.getSettings()
    const now = new Date()
    const next: GroupGuardMessageSettingsRecord = {
      ...current,
      messages: resolveGroupGuardMessages(this.defaults.messages),
      updatedAt: now,
    }
    await this.ctx.database.set(GROUP_GUARD_MESSAGE_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      messages: next.messages,
      updatedAt: now,
    })
    return next
  }

  async getMessages() {
    return (await this.getSettings()).messages
  }

  private async createDefaultRecord() {
    const now = new Date()
    const record: GroupGuardMessageSettingsRecord = {
      id: DEFAULT_SETTINGS_ID,
      messages: resolveGroupGuardMessages(this.defaults.messages),
      createdAt: now,
      updatedAt: now,
    }
    try {
      await this.ctx.database.create(GROUP_GUARD_MESSAGE_SETTINGS_TABLE, record)
    } catch (error) {
      const [existing] = await this.ctx.database.get(GROUP_GUARD_MESSAGE_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
      if (existing) {
        return normalizeRecord(existing, this.defaults)
      }
      throw error
    }
    return record
  }
}

function normalizeRecord(
  record: GroupGuardMessageSettingsRecord,
  defaults: GroupGuardMessageSettings,
): GroupGuardMessageSettingsRecord {
  return {
    ...record,
    messages: resolveGroupGuardMessages(isObject(record.messages) ? record.messages : defaults.messages),
  }
}

function normalizeInput(
  input: GroupGuardMessageSettingsInput,
  current: GroupGuardMessageSettings,
  defaults: GroupGuardMessageSettings,
): Partial<GroupGuardMessageSettings> {
  const messages = normalizeMessages(input.messages)
  if (!messages) {
    return {}
  }
  return {
    messages: resolveGroupGuardMessages({
      ...defaults.messages,
      ...current.messages,
      ...messages,
    }),
  }
}

function normalizeMessages(value: unknown) {
  if (!isObject(value)) {
    return undefined
  }
  const result: Partial<StuhelperGroupGuardMessageConfig> = {}
  for (const key of GROUP_GUARD_MESSAGE_KEYS) {
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
