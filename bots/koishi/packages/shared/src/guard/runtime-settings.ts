import type { Context } from 'koishi'

import { type SettingsReadCache, settingsReadCacheFor } from '../settings-cache'

export const ADMISSION_RUNTIME_SETTINGS_TABLE = 'stuhelper_admission_runtime_settings'
const DEFAULT_SETTINGS_ID = 'default'

declare module 'koishi' {
  interface Tables {
    [ADMISSION_RUNTIME_SETTINGS_TABLE]: AdmissionRuntimeSettingsRecord
  }
}

export interface AdmissionRuntimeSettings {
  actionStreamEnabled: boolean
  publicCommandsEnabled: boolean
  adminCommandsEnabled: boolean
  admissionCommandsEnabled: boolean
  moderationEnabled: boolean
  freshmanForwardEnabled: boolean
  fallbackScanEnabled: boolean
  reminderGroupEnabled: boolean
  reminderDirectEnabled: boolean
}

export interface AdmissionRuntimeSettingsRecord extends AdmissionRuntimeSettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export type AdmissionRuntimeSettingsInput = Partial<AdmissionRuntimeSettings>

export const DEFAULT_ADMISSION_RUNTIME_SETTINGS: AdmissionRuntimeSettings = {
  actionStreamEnabled: true,
  publicCommandsEnabled: false,
  adminCommandsEnabled: true,
  admissionCommandsEnabled: true,
  moderationEnabled: false,
  freshmanForwardEnabled: false,
  fallbackScanEnabled: true,
  reminderGroupEnabled: true,
  reminderDirectEnabled: false,
}

export function registerAdmissionRuntimeSettingsModel(ctx: Context) {
  ctx.model.extend(ADMISSION_RUNTIME_SETTINGS_TABLE, {
    id: 'string',
    actionStreamEnabled: 'boolean',
    publicCommandsEnabled: 'boolean',
    adminCommandsEnabled: 'boolean',
    admissionCommandsEnabled: 'boolean',
    moderationEnabled: 'boolean',
    freshmanForwardEnabled: 'boolean',
    fallbackScanEnabled: 'boolean',
    reminderGroupEnabled: 'boolean',
    reminderDirectEnabled: 'boolean',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class AdmissionRuntimeSettingsStore {
  private readonly cache: SettingsReadCache<AdmissionRuntimeSettingsRecord>

  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: AdmissionRuntimeSettings,
  ) {
    this.cache = settingsReadCacheFor(ctx.database, ADMISSION_RUNTIME_SETTINGS_TABLE)
  }

  async getSettings(): Promise<AdmissionRuntimeSettingsRecord> {
    return this.cache.read(() => this.loadSettings())
  }

  private async loadSettings(): Promise<AdmissionRuntimeSettingsRecord> {
    const [record] = await this.ctx.database.get(ADMISSION_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
    if (record) {
      return normalizeRecord(record, this.defaults)
    }
    return this.createDefaultRecord()
  }

  async saveSettings(input: AdmissionRuntimeSettingsInput) {
    const current = await this.getSettings()
    const now = new Date()
    const changes = normalizeInput(input)
    const next = {
      ...current,
      ...changes,
      updatedAt: now,
    }
    if (!next.reminderGroupEnabled && !next.reminderDirectEnabled) {
      throw new Error('at least one admission reminder delivery channel must be enabled')
    }
    await this.ctx.database.set(ADMISSION_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return this.cache.write(next)
  }

  async isPublicCommandsEnabled() {
    return (await this.getSettings()).publicCommandsEnabled
  }

  async isAdminCommandsEnabled() {
    return (await this.getSettings()).adminCommandsEnabled
  }

  async isActionStreamEnabled() {
    return (await this.getSettings()).actionStreamEnabled
  }

  async isAdmissionCommandsEnabled() {
    return (await this.getSettings()).admissionCommandsEnabled
  }

  async isModerationEnabled() {
    return (await this.getSettings()).moderationEnabled
  }

  async isFreshmanForwardEnabled() {
    return (await this.getSettings()).freshmanForwardEnabled
  }

  async isFallbackScanEnabled() {
    return (await this.getSettings()).fallbackScanEnabled
  }

  async getAdmissionReminderDeliveryConfig() {
    const settings = await this.getSettings()
    return {
      groupEnabled: settings.reminderGroupEnabled,
      directEnabled: settings.reminderDirectEnabled,
    }
  }

  private async createDefaultRecord() {
    const now = new Date()
    const record: AdmissionRuntimeSettingsRecord = {
      id: DEFAULT_SETTINGS_ID,
      ...this.defaults,
      createdAt: now,
      updatedAt: now,
    }
    await this.ctx.database.create(ADMISSION_RUNTIME_SETTINGS_TABLE, record)
    return record
  }
}

function normalizeRecord(
  record: AdmissionRuntimeSettingsRecord,
  defaults: AdmissionRuntimeSettings,
): AdmissionRuntimeSettingsRecord {
  const reminderGroupEnabled = booleanOrDefault(record.reminderGroupEnabled, defaults.reminderGroupEnabled)
  const reminderDirectEnabled = booleanOrDefault(record.reminderDirectEnabled, defaults.reminderDirectEnabled)
  return {
    ...record,
    actionStreamEnabled: booleanOrDefault(record.actionStreamEnabled, defaults.actionStreamEnabled),
    publicCommandsEnabled: booleanOrDefault(record.publicCommandsEnabled, defaults.publicCommandsEnabled),
    adminCommandsEnabled: booleanOrDefault(record.adminCommandsEnabled, defaults.adminCommandsEnabled),
    admissionCommandsEnabled: booleanOrDefault(record.admissionCommandsEnabled, defaults.admissionCommandsEnabled),
    moderationEnabled: booleanOrDefault(record.moderationEnabled, defaults.moderationEnabled),
    freshmanForwardEnabled: booleanOrDefault(record.freshmanForwardEnabled, defaults.freshmanForwardEnabled),
    fallbackScanEnabled: booleanOrDefault(record.fallbackScanEnabled, defaults.fallbackScanEnabled),
    reminderGroupEnabled: reminderGroupEnabled || !reminderDirectEnabled,
    reminderDirectEnabled,
  }
}

function normalizeInput(input: AdmissionRuntimeSettingsInput) {
  return Object.fromEntries(
    Object.entries(input).filter(([, value]) => typeof value === 'boolean'),
  ) as AdmissionRuntimeSettingsInput
}

function booleanOrDefault(value: unknown, fallback: boolean) {
  return typeof value === 'boolean' ? value : fallback
}
