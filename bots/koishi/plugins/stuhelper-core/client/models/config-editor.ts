import type {
  BindingFormState,
  PolicyFormState,
  TemplateFormState,
} from './config-forms'

export const DISCARD_CHANGES_MESSAGE = '当前有未保存的改动，继续会丢失这些修改。'

export function cloneTemplateForm(state: TemplateFormState): TemplateFormState {
  return { ...state }
}

export function cloneBindingForm(state: BindingFormState): BindingFormState {
  return { ...state }
}

export function clonePolicyForm(state: PolicyFormState): PolicyFormState {
  return { ...state }
}

export function isTemplateFormDirty(state: TemplateFormState, snapshot: TemplateFormState | null) {
  return !snapshot || !isSameTemplateForm(state, snapshot)
}

export function isBindingFormDirty(state: BindingFormState, snapshot: BindingFormState | null) {
  return !snapshot || !isSameBindingForm(state, snapshot)
}

export function isPolicyFormDirty(state: PolicyFormState, snapshot: PolicyFormState | null) {
  return !snapshot || !isSamePolicyForm(state, snapshot)
}

export function confirmDiscardChanges(
  dirty: boolean,
  confirm: (message: string) => boolean,
) {
  return !dirty || confirm(DISCARD_CHANGES_MESSAGE)
}

function isSameTemplateForm(left: TemplateFormState, right: TemplateFormState) {
  return left.id === right.id
    && left.name === right.name
    && left.muteDurationSeconds === right.muteDurationSeconds
    && left.kickAfterMinutes === right.kickAfterMinutes
    && left.reminderTemplate === right.reminderTemplate
    && left.exemptUsersText === right.exemptUsersText
    && left.enabled === right.enabled
}

function isSameBindingForm(left: BindingFormState, right: BindingFormState) {
  return left.platform === right.platform
    && left.guildId === right.guildId
    && left.templateId === right.templateId
    && left.note === right.note
    && left.enabled === right.enabled
}

function isSamePolicyForm(left: PolicyFormState, right: PolicyFormState) {
  return left.commandId === right.commandId
    && left.minAuthority === right.minAuthority
    && left.rolesText === right.rolesText
}
