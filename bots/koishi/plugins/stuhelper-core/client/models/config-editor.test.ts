import assert from 'node:assert/strict'
import test from 'node:test'

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
    platform: 'onebot',
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
