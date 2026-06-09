import assert from 'node:assert/strict'
import test from 'node:test'

import { parseAdminRuntimeSettingsInput } from './admin-runtime-settings-api'

test('parseAdminRuntimeSettingsInput accepts admin message templates', () => {
  const input = parseAdminRuntimeSettingsInput({
    messages: {
      guardPendingMembersHeader: '自定义待认证：',
      commandAccessDenied: '自定义：无权限。',
    },
  })

  assert.deepEqual(input, {
    messages: {
      guardPendingMembersHeader: '自定义待认证：',
      commandAccessDenied: '自定义：无权限。',
    },
  })
})

test('parseAdminRuntimeSettingsInput rejects native-plugin-only and unknown fields', () => {
  assert.throws(
    () => parseAdminRuntimeSettingsInput({
      platform: { baseUrl: 'http://127.0.0.1:8080' },
    }),
    /unsupported field: platform/,
  )
  assert.throws(
    () => parseAdminRuntimeSettingsInput({
      messages: {
        unknownMessage: 'x',
      },
    }),
    /unsupported field: unknownMessage/,
  )
  assert.throws(
    () => parseAdminRuntimeSettingsInput({
      messages: {
        commandAccessDenied: 1,
      },
    }),
    /admin message commandAccessDenied must be a string/,
  )
  assert.throws(
    () => parseAdminRuntimeSettingsInput({
      messages: {
        commandAccessDenied: 'x'.repeat(2001),
      },
    }),
    /admin message commandAccessDenied must be at most 2000 characters/,
  )
})
