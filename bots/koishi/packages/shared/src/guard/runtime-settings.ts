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
  admissionCommandsEnabled: boolean
  moderationEnabled: boolean
  freshmanForwardEnabled: boolean
  fallbackScanEnabled: boolean
}

export interface AdmissionRuntimeSettingsRecord extends AdmissionRuntimeSettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export type AdmissionRuntimeSettingsInput = Partial<AdmissionRuntimeSettings>

export function registerAdmissionRuntimeSettingsModel(ctx: Context) {
  ctx.model.extend(ADMISSION_RUNTIME_SETTINGS_TABLE, {
    id: 'string',
    actionStreamEnabled: 'boolean',
    publicCommandsEnabled: 'boolean',
    admissionCommandsEnabled: 'boolean',
    moderationEnabled: 'boolean',
    freshmanForwardEnabled: 'boolean',
    fallbackScanEnabled: 'boolean',
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
    admissionCommandsEnabled: booleanOrDefault(record.admissionCommandsEnabled, defaults.admissionCommandsEnabled),
    moderationEnabled: booleanOrDefault(record.moderationEnabled, defaults.moderationEnabled),
    freshmanForwardEnabled: booleanOrDefault(record.freshmanForwardEnabled, defaults.freshmanForwardEnabled),
    fallbackScanEnabled: booleanOrDefault(record.fallbackScanEnabled, defaults.fallbackScanEnabled),
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
