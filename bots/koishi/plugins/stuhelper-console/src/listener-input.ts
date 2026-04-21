import type {
  StuhelperCommandPolicyInput,
  StuhelperGuardBatchActionInput,
  StuhelperGuardBindingInput,
  StuhelperGuardTemplateInput,
  StuhelperKeywordRuleInput,
  StuhelperMemberRoleInput,
  StuhelperReviewActionInput,
} from './console-types'

type EnumValues<T extends string> = readonly T[]

export function parseGuardBatchActionInput(input: unknown): StuhelperGuardBatchActionInput {
  const record = requireRecord(input, 'guard action')
  return {
    action: readEnum(record.action, ['mute', 'unmute', 'kick', 'set-role', 'unset-role'], 'action'),
    memberIds: readStringArray(record.memberIds, 'memberIds'),
    seconds: readOptionalNumber(record.seconds, 'seconds'),
    reason: readString(record.reason, 'reason'),
    roleId: readOptionalString(record.roleId, 'roleId'),
    permanent: readOptionalBoolean(record.permanent, 'permanent'),
  }
}

export function parseReviewActionInput(input: unknown): StuhelperReviewActionInput {
  const record = requireRecord(input, 'review action')
  return {
    reviewId: readString(record.reviewId, 'reviewId'),
    action: readEnum(record.action, ['execute', 'reject'], 'action'),
    note: readOptionalString(record.note, 'note'),
  }
}

export function parseKeywordRuleInput(input: unknown): StuhelperKeywordRuleInput {
  const record = requireRecord(input, 'keyword rule')
  return {
    id: readString(record.id, 'id'),
    guildId: readString(record.guildId, 'guildId'),
    pattern: readString(record.pattern, 'pattern'),
    matchMode: readEnum(record.matchMode, ['includes', 'regex'], 'matchMode'),
    action: readEnum(record.action, ['warn', 'delete', 'mute', 'review'], 'action'),
    enabled: readBoolean(record.enabled, 'enabled'),
    muteSeconds: readNumber(record.muteSeconds, 'muteSeconds'),
    note: readOptionalNullableString(record.note, 'note'),
  }
}

export function parseMemberRoleInput(input: unknown): StuhelperMemberRoleInput {
  const record = requireRecord(input, 'member roles')
  return {
    guildId: readString(record.guildId, 'guildId'),
    memberId: readString(record.memberId, 'memberId'),
    roles: readStringArray(record.roles, 'roles'),
  }
}

export function parseCommandPolicyInput(input: unknown): StuhelperCommandPolicyInput {
  const record = requireRecord(input, 'command policy')
  return {
    commandId: readString(record.commandId, 'commandId'),
    minAuthority: readNumber(record.minAuthority, 'minAuthority'),
    roles: readStringArray(record.roles, 'roles'),
  }
}

export function parseGuardTemplateInput(input: unknown): StuhelperGuardTemplateInput {
  const record = requireRecord(input, 'guard template')
  return {
    id: readString(record.id, 'id'),
    name: readString(record.name, 'name'),
    muteDurationSeconds: readNumber(record.muteDurationSeconds, 'muteDurationSeconds'),
    kickAfterMinutes: readNumber(record.kickAfterMinutes, 'kickAfterMinutes'),
    reminderTemplate: readString(record.reminderTemplate, 'reminderTemplate'),
    exemptUsers: readStringArray(record.exemptUsers, 'exemptUsers'),
    enabled: readBoolean(record.enabled, 'enabled'),
  }
}

export function parseGuardBindingInput(input: unknown): StuhelperGuardBindingInput {
  const record = requireRecord(input, 'guard binding')
  return {
    platform: readString(record.platform, 'platform'),
    guildId: readString(record.guildId, 'guildId'),
    templateId: readString(record.templateId, 'templateId'),
    enabled: readBoolean(record.enabled, 'enabled'),
    note: readOptionalNullableString(record.note, 'note'),
  }
}

function requireRecord(input: unknown, label: string) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as Record<string, unknown>
}

function readString(value: unknown, field: string): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${field} must be a non-empty string`)
  }
  return value
}

function readOptionalString(value: unknown, field: string): string | undefined {
  if (value === undefined) return undefined
  if (typeof value !== 'string') {
    throw new Error(`${field} must be a string`)
  }
  return value
}

function readOptionalNullableString(value: unknown, field: string): string | null | undefined {
  if (value === undefined) return undefined
  if (value === null) return null
  if (typeof value !== 'string') {
    throw new Error(`${field} must be a string or null`)
  }
  return value
}

function readStringArray(value: unknown, field: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== 'string')) {
    throw new Error(`${field} must be a string array`)
  }
  return [...value]
}

function readNumber(value: unknown, field: string): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`${field} must be a valid number`)
  }
  return value
}

function readOptionalNumber(value: unknown, field: string): number | undefined {
  if (value === undefined) return undefined
  return readNumber(value, field)
}

function readBoolean(value: unknown, field: string): boolean {
  if (typeof value !== 'boolean') {
    throw new Error(`${field} must be a boolean`)
  }
  return value
}

function readOptionalBoolean(value: unknown, field: string): boolean | undefined {
  if (value === undefined) return undefined
  return readBoolean(value, field)
}

function readEnum<T extends string>(value: unknown, values: EnumValues<T>, field: string) {
  if (typeof value !== 'string' || !values.includes(value as T)) {
    throw new Error(`${field} must be one of: ${values.join(', ')}`)
  }
  return value as T
}
