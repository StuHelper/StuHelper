import assert from 'node:assert/strict'
import test from 'node:test'

import {
  assignBindingFormState,
  assignPolicyFormState,
  assignTemplateFormState,
  createBindingForm,
  createPolicyForm,
  createTemplateForm,
} from './config-forms'
import {
  DISCARD_CHANGES_MESSAGE,
  cloneBindingForm,
  clonePolicyForm,
  cloneTemplateForm,
  confirmDiscardChanges,
  isBindingFormDirty,
  isPolicyFormDirty,
  isTemplateFormDirty,
} from './config-editor'

test('config editor form snapshots detect dirty state per form', () => {
  const template = cloneTemplateForm({
    id: 'tpl-1',
    name: '默认模板',
    muteDurationSeconds: 1800,
    kickAfterMinutes: 30,
    reminderTemplate: '请先完成认证',
    exemptUsersText: '1001',
    enabled: true,
  })
  const binding = cloneBindingForm({
    platform: 'qq',
    guildId: '1001',
    templateId: 'tpl-1',
    note: '主群',
    enabled: true,
  })
  const policy = clonePolicyForm({
    commandId: 'report',
    minAuthority: 3,
    rolesText: 'admin',
  })

  assert.equal(isTemplateFormDirty(template, template), false)
  assert.equal(isBindingFormDirty(binding, binding), false)
  assert.equal(isPolicyFormDirty(policy, policy), false)

  assert.equal(isTemplateFormDirty({ ...template, name: '升级模板' }, template), true)
  assert.equal(isBindingFormDirty({ ...binding, note: '分群' }, binding), true)
  assert.equal(isPolicyFormDirty({ ...policy, minAuthority: 4 }, policy), true)
})

test('config editor default new-record snapshots start clean', () => {
  const template = createTemplateForm()
  const binding = createBindingForm()
  const policy = createPolicyForm()

  assert.equal(isTemplateFormDirty(template, cloneTemplateForm(template)), false)
  assert.equal(isBindingFormDirty(binding, cloneBindingForm(binding)), false)
  assert.equal(isPolicyFormDirty(policy, clonePolicyForm(policy)), false)
  assert.equal(binding.platform, 'qq')

  assert.equal(isTemplateFormDirty({ ...template, id: 'draft' }, template), true)
  assert.equal(isBindingFormDirty({ ...binding, guildId: '1001' }, binding), true)
  assert.equal(isPolicyFormDirty({ ...policy, commandId: 'report' }, policy), true)
})

test('config editor can restore form state from snapshots after discard', () => {
  const template = createTemplateForm()
  const binding = createBindingForm()
  const policy = createPolicyForm()
  const templateSnapshot = cloneTemplateForm(template)
  const bindingSnapshot = cloneBindingForm(binding)
  const policySnapshot = clonePolicyForm(policy)

  template.id = 'draft'
  binding.guildId = '1001'
  policy.commandId = 'report'

  assignTemplateFormState(template, templateSnapshot)
  assignBindingFormState(binding, bindingSnapshot)
  assignPolicyFormState(policy, policySnapshot)

  assert.equal(isTemplateFormDirty(template, templateSnapshot), false)
  assert.equal(isBindingFormDirty(binding, bindingSnapshot), false)
  assert.equal(isPolicyFormDirty(policy, policySnapshot), false)
})

test('confirmDiscardChanges skips confirmation when the form is clean', () => {
  let calls = 0
  const allowed = confirmDiscardChanges(false, () => {
    calls += 1
    return false
  })

  assert.equal(allowed, true)
  assert.equal(calls, 0)
})

test('confirmDiscardChanges delegates to the confirm callback when dirty', () => {
  let message = ''
  const allowed = confirmDiscardChanges(true, (value) => {
    message = value
    return true
  })

  assert.equal(allowed, true)
  assert.equal(message, DISCARD_CHANGES_MESSAGE)
})
