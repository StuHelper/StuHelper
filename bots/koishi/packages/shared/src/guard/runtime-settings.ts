import type { Context } from 'koishi'

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
  fallbackScanEnabled: boolean
  reminderGroupEnabled: boolean
  reminderDirectEnabled: boolean
  timeCodeReminderEnabled: boolean
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
  fallbackScanEnabled: true,
  reminderGroupEnabled: true,
  reminderDirectEnabled: false,
  timeCodeReminderEnabled: true,
}

export function registerAdmissionRuntimeSettingsModel(ctx: Context) {
  ctx.model.extend(ADMISSION_RUNTIME_SETTINGS_TABLE, {
    id: 'string',
    actionStreamEnabled: 'boolean',
    publicCommandsEnabled: 'boolean',
    adminCommandsEnabled: 'boolean',
    admissionCommandsEnabled: 'boolean',
    moderationEnabled: 'boolean',
    fallbackScanEnabled: 'boolean',
    reminderGroupEnabled: 'boolean',
    reminderDirectEnabled: 'boolean',
    timeCodeReminderEnabled: 'boolean',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class AdmissionRuntimeSettingsStore {
  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: AdmissionRuntimeSettings,
  ) {}

  async getSettings(): Promise<AdmissionRuntimeSettingsRecord> {
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
    await this.ctx.database.set(ADMISSION_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return next
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

  async isTimeCodeReminderEnabled() {
    return (await this.getSettings()).timeCodeReminderEnabled
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
  return {
    ...record,
    actionStreamEnabled: booleanOrDefault(record.actionStreamEnabled, defaults.actionStreamEnabled),
    publicCommandsEnabled: booleanOrDefault(record.publicCommandsEnabled, defaults.publicCommandsEnabled),
    adminCommandsEnabled: booleanOrDefault(record.adminCommandsEnabled, defaults.adminCommandsEnabled),
    admissionCommandsEnabled: booleanOrDefault(record.admissionCommandsEnabled, defaults.admissionCommandsEnabled),
    moderationEnabled: booleanOrDefault(record.moderationEnabled, defaults.moderationEnabled),
    fallbackScanEnabled: booleanOrDefault(record.fallbackScanEnabled, defaults.fallbackScanEnabled),
    reminderGroupEnabled: booleanOrDefault(record.reminderGroupEnabled, defaults.reminderGroupEnabled),
    reminderDirectEnabled: booleanOrDefault(record.reminderDirectEnabled, defaults.reminderDirectEnabled),
    timeCodeReminderEnabled: booleanOrDefault(record.timeCodeReminderEnabled, defaults.timeCodeReminderEnabled),
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
