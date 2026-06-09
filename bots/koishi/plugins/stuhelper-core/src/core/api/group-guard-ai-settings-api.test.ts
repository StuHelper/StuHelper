import assert from 'node:assert/strict'
import test from 'node:test'

import {
  parseGroupGuardAISettingsInput,
  toPublicAISettings,
} from './group-guard-ai-settings-api'

test('parseGroupGuardAISettingsInput accepts runtime ai settings and new key replacement', () => {
  const input = parseGroupGuardAISettingsInput({
    enabled: true,
    endpoint: ' https://example.test/review ',
    model: ' gpt-test ',
    newApiKey: ' test-key ',
  })

  assert.deepEqual(input, {
    enabled: true,
    endpoint: 'https://example.test/review',
    model: 'gpt-test',
    apiKey: 'test-key',
  })
})

test('parseGroupGuardAISettingsInput accepts explicit api key clear', () => {
  const input = parseGroupGuardAISettingsInput({
    clearApiKey: true,
  })

  assert.deepEqual(input, { apiKey: '' })
})

test('parseGroupGuardAISettingsInput rejects public, native and unknown fields', () => {
  for (const field of ['apiKey', 'apiKeyConfigured', 'apiKeyMasked', 'id', 'createdAt', 'platform', 'unknown']) {
    assert.throws(
      () => parseGroupGuardAISettingsInput({ [field]: 'value' }),
      new RegExp(`unsupported field: ${field}`),
    )
  }
  assert.throws(
    () => parseGroupGuardAISettingsInput({ enabled: 'true' }),
    /enabled must be a boolean/,
  )
  assert.throws(
    () => parseGroupGuardAISettingsInput({ endpoint: 123 }),
    /endpoint must be a string/,
  )
  assert.throws(
    () => parseGroupGuardAISettingsInput({ model: 'x'.repeat(257) }),
    /model must be at most 256 characters/,
  )
  assert.throws(
    () => parseGroupGuardAISettingsInput({ newApiKey: 'x'.repeat(4097) }),
    /newApiKey must be at most 4096 characters/,
  )
  assert.throws(
    () => parseGroupGuardAISettingsInput({ clearApiKey: 'yes' }),
    /clearApiKey must be a boolean/,
  )
  assert.throws(
    () => parseGroupGuardAISettingsInput({ clearApiKey: true, newApiKey: 'test-key' }),
    /不能同时清除和替换 API Key/,
  )
})

test('toPublicAISettings masks api key and never returns plaintext', () => {
  const data = toPublicAISettings({
    id: 'default',
    enabled: true,
    endpoint: 'https://example.test/review',
    apiKey: 'sk-test-secret',
    model: 'gpt-test',
    createdAt: new Date('2026-06-09T10:00:00.000Z'),
    updatedAt: new Date('2026-06-09T10:01:00.000Z'),
  })

  assert.equal(data.apiKeyConfigured, true)
  assert.equal(data.apiKeyMasked, 'sk-t...cret')
  assert.equal('apiKey' in data, false)
})
