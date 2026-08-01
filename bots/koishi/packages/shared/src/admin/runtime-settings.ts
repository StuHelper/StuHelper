import type { Context } from 'koishi'

import {
  DEFAULT_ADMIN_MESSAGES,
  resolveAdminMessages,
} from '../message-template'
import type { StuhelperAdminMessageConfig } from '../types/index'

export const ADMIN_RUNTIME_SETTINGS_TABLE = 'stuhelper_admin_runtime_settings'

const DEFAULT_SETTINGS_ID = 'default'

export const ADMIN_MESSAGE_KEYS = Object.freeze([
  'guardStatusCommandDescription',
  'guardWarningCommandDescription',
  'guardReviewListCommandDescription',
  'guardBatchMuteCommandDescription',
  'guardKickReviewCommandDescription',
  'guardBlockReviewCommandDescription',
  'freshmanViewCommandDescription',
  'freshmanApproveCommandDescription',
  'freshmanRejectCommandDescription',
  'freshmanBlacklistReleaseCommandDescription',
  'commandAccessDenied',
  'adminCommandsDisabled',
  'guardGuildContextRequired',
  'guardWarningMissingContext',
  'guardBatchMuteGroupOnly',
  'guardBatchMuteInvalidPayload',
  'guardBatchMuteNoTargets',
  'guardBatchMuteSuccess',
  'guardBatchMuteEventSummary',
  'guardBatchMuteEventReason',
  'guardReviewRequestMissingArgs',
  'guardReviewRequestSuccess',
  'guardReviewRequestEventSummary',
  'guardReviewKickActionLabel',
  'guardReviewKickAndBlockActionLabel',
  'guardPendingMembersEmpty',
  'guardPendingMembersHeader',
  'guardPendingMemberLine',
  'guardWarningCounterEmpty',
  'guardWarningCounterNoReason',
  'guardWarningCounterSummary',
  'guardPendingReviewsEmpty',
  'guardPendingReviewsHeader',
  'guardPendingReviewLine',
  'freshmanManagementGroupOnly',
  'freshmanMissingApplicationID',
  'freshmanApproveInvalidFormat',
  'freshmanApproveInvalidExtension',
  'freshmanRejectInvalidFormat',
  'freshmanBlacklistReleaseInvalidFormat',
  'freshmanApproveSuccess',
  'freshmanApproveSuccessWithExtension',
  'freshmanRejectSuccess',
  'freshmanBlacklistScopeGlobal',
  'freshmanBlacklistScopeGuild',
  'freshmanBlacklistReleaseSuccess',
  'freshmanApplicationSummary',
  'freshmanApplicationDepartmentFallback',
  'freshmanApplicationExpiryFallback',
  'freshmanCommandFailed',
  'freshmanOperatorQQUnbound',
  'freshmanOperatorForbidden',
  'freshmanManagementGuildForbidden',
  'freshmanBackendForbidden',
] as const satisfies readonly (keyof StuhelperAdminMessageConfig)[])

declare module 'koishi' {
  interface Tables {
    [ADMIN_RUNTIME_SETTINGS_TABLE]: AdminRuntimeSettingsRecord
  }
}

export interface AdminRuntimeSettings {
  messages: StuhelperAdminMessageConfig
}

export interface AdminRuntimeSettingsRecord extends AdminRuntimeSettings {
  id: string
  createdAt: Date
  updatedAt: Date
}

export interface AdminRuntimeSettingsInput {
  messages?: Partial<StuhelperAdminMessageConfig>
}

export const DEFAULT_ADMIN_RUNTIME_SETTINGS: AdminRuntimeSettings = {
  messages: DEFAULT_ADMIN_MESSAGES,
}

export function registerAdminRuntimeSettingsModel(ctx: Context) {
  ctx.model.extend(ADMIN_RUNTIME_SETTINGS_TABLE, {
    id: 'string',
    messages: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

export class AdminRuntimeSettingsStore {
  constructor(
    private readonly ctx: Pick<Context, 'database'>,
    private readonly defaults: AdminRuntimeSettings = DEFAULT_ADMIN_RUNTIME_SETTINGS,
  ) {}

  async getSettings(): Promise<AdminRuntimeSettingsRecord> {
    const [record] = await this.ctx.database.get(ADMIN_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
    if (record) {
      return normalizeRecord(record, this.defaults)
    }
    return this.createDefaultRecord()
  }

  async saveSettings(input: AdminRuntimeSettingsInput) {
    const current = await this.getSettings()
    const changes = normalizeInput(input, current, this.defaults)
    const now = new Date()
    const next: AdminRuntimeSettingsRecord = {
      ...current,
      ...changes,
      updatedAt: now,
    }
    await this.ctx.database.set(ADMIN_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
      ...changes,
      updatedAt: now,
    })
    return next
  }

  async resetSettings() {
    const current = await this.getSettings()
    const now = new Date()
    const next: AdminRuntimeSettingsRecord = {
      ...current,
      ...this.defaults,
      messages: resolveAdminMessages(this.defaults.messages),
      updatedAt: now,
    }
    await this.ctx.database.set(ADMIN_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID }, {
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
    const record: AdminRuntimeSettingsRecord = {
      id: DEFAULT_SETTINGS_ID,
      messages: resolveAdminMessages(this.defaults.messages),
      createdAt: now,
      updatedAt: now,
    }
    try {
      await this.ctx.database.create(ADMIN_RUNTIME_SETTINGS_TABLE, record)
    } catch (error) {
      const [existing] = await this.ctx.database.get(ADMIN_RUNTIME_SETTINGS_TABLE, { id: DEFAULT_SETTINGS_ID })
      if (existing) {
        return normalizeRecord(existing, this.defaults)
      }
      throw error
    }
    return record
  }
}

function normalizeRecord(
  record: AdminRuntimeSettingsRecord,
  defaults: AdminRuntimeSettings,
): AdminRuntimeSettingsRecord {
  return {
    ...record,
    messages: resolveAdminMessages(isObject(record.messages) ? record.messages : defaults.messages),
  }
}

function normalizeInput(
  input: AdminRuntimeSettingsInput,
  current: AdminRuntimeSettings,
  defaults: AdminRuntimeSettings,
): Partial<AdminRuntimeSettings> {
  const messages = normalizeMessages(input.messages)
  if (!messages) {
    return {}
  }
  return {
    messages: resolveAdminMessages({
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
  const result: Partial<StuhelperAdminMessageConfig> = {}
  for (const key of ADMIN_MESSAGE_KEYS) {
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
