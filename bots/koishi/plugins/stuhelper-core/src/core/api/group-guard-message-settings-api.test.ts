import assert from 'node:assert/strict'
import test from 'node:test'

import { parseGroupGuardMessageSettingsInput } from './group-guard-message-settings-api'

test('parseGroupGuardMessageSettingsInput accepts group guard message templates', () => {
  const input = parseGroupGuardMessageSettingsInput({
    messages: {
      admissionReminder: '请认证：{authURL}',
      reportHighRisk: '高风险举报已进入复核。',
    },
  })

  assert.deepEqual(input, {
    messages: {
      admissionReminder: '请认证：{authURL}',
      reportHighRisk: '高风险举报已进入复核。',
    },
  })
})

test('parseGroupGuardMessageSettingsInput rejects native-plugin-only and unknown fields', () => {
  assert.throws(
    () => parseGroupGuardMessageSettingsInput({
      platform: { serviceToken: 'secret' },
    }),
    /unsupported field: platform/,
  )
  assert.throws(
    () => parseGroupGuardMessageSettingsInput({
      messages: {
        unknownMessage: 'x',
      },
    }),
    /unsupported field: unknownMessage/,
  )
  assert.throws(
    () => parseGroupGuardMessageSettingsInput({
      messages: {
        admissionReminder: 1,
      },
    }),
    /group guard message admissionReminder must be a string/,
  )
  assert.throws(
    () => parseGroupGuardMessageSettingsInput({
      messages: {
        admissionReminder: 'x'.repeat(3001),
      },
    }),
    /group guard message admissionReminder must be at most 3000 characters/,
  )
})
