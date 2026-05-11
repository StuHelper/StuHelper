import type { ConfigGovernancePageData } from '../page-types'

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
    platform: 'onebot',
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
