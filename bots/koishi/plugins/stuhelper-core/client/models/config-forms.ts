import type { ConfigGovernancePageData } from '../page-types'

export const MAX_COMMAND_ID_LENGTH = 128
export const MAX_ROLE_ID_LENGTH = 64
export const MAX_TEMPLATE_ID_LENGTH = 64
export const MAX_TEMPLATE_NAME_LENGTH = 128
export const MAX_REMINDER_TEMPLATE_LENGTH = 2000
export const MAX_EXEMPT_USER_COUNT = 500
export const MAX_NOTE_LENGTH = 512
export const MAX_MUTE_DURATION_SECONDS = 30 * 24 * 3600
export const MAX_KICK_AFTER_MINUTES = 30 * 24 * 60
export const MAX_MIN_AUTHORITY = 5

export interface TemplateFormState {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsersText: string
  enabled: boolean
}

export interface BindingFormState {
  platform: string
  guildId: string
  templateId: string
  note: string
  enabled: boolean
}

export interface PolicyFormState {
  commandId: string
  minAuthority: number
  rolesText: string
}

export function createTemplateForm(): TemplateFormState {
  return {
    id: '',
    name: '',
    muteDurationSeconds: 1800,
    kickAfterMinutes: 30,
    reminderTemplate: '',
    exemptUsersText: '',
    enabled: true,
  }
}

export function createBindingForm(): BindingFormState {
  return {
    platform: 'qq',
    guildId: '',
    templateId: '',
    note: '',
    enabled: true,
  }
}

export function createPolicyForm(): PolicyFormState {
  return {
    commandId: '',
    minAuthority: 3,
    rolesText: '',
  }
}

export function assignTemplateFormState(state: TemplateFormState, source: TemplateFormState) {
  state.id = source.id
  state.name = source.name
  state.muteDurationSeconds = source.muteDurationSeconds
  state.kickAfterMinutes = source.kickAfterMinutes
  state.reminderTemplate = source.reminderTemplate
  state.exemptUsersText = source.exemptUsersText
  state.enabled = source.enabled
}

export function assignBindingFormState(state: BindingFormState, source: BindingFormState) {
  state.platform = source.platform
  state.guildId = source.guildId
  state.templateId = source.templateId
  state.note = source.note
  state.enabled = source.enabled
}

export function assignPolicyFormState(state: PolicyFormState, source: PolicyFormState) {
  state.commandId = source.commandId
  state.minAuthority = source.minAuthority
  state.rolesText = source.rolesText
}

export function assignTemplateForm(
  state: TemplateFormState,
  item: ConfigGovernancePageData['templates'][number],
) {
  state.id = item.id
  state.name = item.name
  state.muteDurationSeconds = item.muteDurationSeconds
  state.kickAfterMinutes = item.kickAfterMinutes
  state.reminderTemplate = item.reminderTemplate
  state.exemptUsersText = item.exemptUsers.join(', ')
  state.enabled = item.enabled
}

export function assignBindingForm(
  state: BindingFormState,
  item: ConfigGovernancePageData['bindings'][number],
) {
  state.platform = item.platform
  state.guildId = item.guildId
  state.templateId = item.templateId
  state.note = item.note || ''
  state.enabled = item.enabled
}

export function assignPolicyForm(
  state: PolicyFormState,
  item: ConfigGovernancePageData['commandPolicies'][number],
) {
  state.commandId = item.commandId
  state.minAuthority = item.minAuthority
  state.rolesText = item.roles.join(', ')
}

export function splitCommaTokens(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

export function validateTemplateForm(state: TemplateFormState): string {
  if (!state.id.trim()) return '模板 ID 不能为空'
  if (state.id.trim().length > MAX_TEMPLATE_ID_LENGTH) {
    return `模板 ID 不能超过 ${MAX_TEMPLATE_ID_LENGTH} 个字符`
  }
  if (!state.name.trim()) return '模板名称不能为空'
  if (state.name.trim().length > MAX_TEMPLATE_NAME_LENGTH) {
    return `模板名称不能超过 ${MAX_TEMPLATE_NAME_LENGTH} 个字符`
  }
  if (!isIntegerInRange(state.muteDurationSeconds, 0, MAX_MUTE_DURATION_SECONDS)) {
    return `禁言时长必须是 0 到 ${MAX_MUTE_DURATION_SECONDS} 之间的整数秒`
  }
  if (!isIntegerInRange(state.kickAfterMinutes, 0, MAX_KICK_AFTER_MINUTES)) {
    return `踢出阈值必须是 0 到 ${MAX_KICK_AFTER_MINUTES} 之间的整数分钟`
  }
  if (!state.reminderTemplate.trim()) return '提醒文案不能为空'
  if (state.reminderTemplate.trim().length > MAX_REMINDER_TEMPLATE_LENGTH) {
    return `提醒文案不能超过 ${MAX_REMINDER_TEMPLATE_LENGTH} 个字符`
  }

  const exemptUsers = splitCommaTokens(state.exemptUsersText)
  if (exemptUsers.length > MAX_EXEMPT_USER_COUNT) {
    return `豁免名单不能超过 ${MAX_EXEMPT_USER_COUNT} 个成员`
  }
  const tooLongExemptUser = exemptUsers.find((item) => item.length > MAX_ROLE_ID_LENGTH)
  if (tooLongExemptUser) {
    return `豁免成员 ${tooLongExemptUser} 不能超过 ${MAX_ROLE_ID_LENGTH} 个字符`
  }
  return ''
}

export function validateBindingForm(state: BindingFormState): string {
  if (!state.platform.trim()) return '平台不能为空'
  if (state.platform.trim().length > MAX_ROLE_ID_LENGTH) {
    return `平台不能超过 ${MAX_ROLE_ID_LENGTH} 个字符`
  }
  if (!state.guildId.trim()) return '群号不能为空'
  if (state.guildId.trim().length > MAX_ROLE_ID_LENGTH) {
    return `群号不能超过 ${MAX_ROLE_ID_LENGTH} 个字符`
  }
  if (!state.templateId.trim()) return '模板不能为空'
  if (state.templateId.trim().length > MAX_TEMPLATE_ID_LENGTH) {
    return `模板 ID 不能超过 ${MAX_TEMPLATE_ID_LENGTH} 个字符`
  }
  if (state.note.trim().length > MAX_NOTE_LENGTH) {
    return `备注不能超过 ${MAX_NOTE_LENGTH} 个字符`
  }
  return ''
}

export function validatePolicyForm(state: PolicyFormState): string {
  if (!state.commandId.trim()) return '命令不能为空'
  if (state.commandId.trim().length > MAX_COMMAND_ID_LENGTH) {
    return `命令不能超过 ${MAX_COMMAND_ID_LENGTH} 个字符`
  }
  if (!isIntegerInRange(state.minAuthority, 0, MAX_MIN_AUTHORITY)) {
    return `最小 authority 必须是 0 到 ${MAX_MIN_AUTHORITY} 之间的整数`
  }

  const roles = splitCommaTokens(state.rolesText)
  if (roles.length > MAX_EXEMPT_USER_COUNT) {
    return `角色白名单不能超过 ${MAX_EXEMPT_USER_COUNT} 个角色`
  }
  const tooLongRole = roles.find((item) => item.length > MAX_ROLE_ID_LENGTH)
  if (tooLongRole) {
    return `角色 ${tooLongRole} 不能超过 ${MAX_ROLE_ID_LENGTH} 个字符`
  }
  return ''
}

function isIntegerInRange(value: number, min: number, max: number): boolean {
  return Number.isInteger(value) && value >= min && value <= max
}
