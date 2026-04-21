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
