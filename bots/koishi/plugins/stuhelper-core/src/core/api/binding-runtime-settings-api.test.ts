import assert from 'node:assert/strict'
import test from 'node:test'

import { parseBindingRuntimeSettingsInput } from './binding-runtime-settings-api'

test('parseBindingRuntimeSettingsInput trims command and accepts binding message templates', () => {
  const input = parseBindingRuntimeSettingsInput({
    command: '  绑定账号  ',
    messages: {
      directOnly: '请私聊',
      missingCode: '请输入 {command} 后面的绑定码',
    },
  })

  assert.deepEqual(input, {
    command: '绑定账号',
    messages: {
      directOnly: '请私聊',
      missingCode: '请输入 {command} 后面的绑定码',
    },
  })
})

test('parseBindingRuntimeSettingsInput rejects native-plugin-only and unknown fields', () => {
  assert.throws(
    () => parseBindingRuntimeSettingsInput({
      command: '',
    }),
    /binding command is required/,
  )
  assert.throws(
    () => parseBindingRuntimeSettingsInput({
      codeTtlMinutes: 10,
    }),
    /unsupported field: codeTtlMinutes/,
  )
  assert.throws(
    () => parseBindingRuntimeSettingsInput({
      messages: {
        unknownMessage: 'x',
      },
    }),
    /unsupported field: unknownMessage/,
  )
})
