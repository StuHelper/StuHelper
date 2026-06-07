import assert from 'node:assert/strict'
import test from 'node:test'

import type { CommandLogRecord } from './log.module'
import {
  redactCommandLogRecord,
  redactSensitiveText,
  redactSensitiveValue,
} from './log-redaction'

test('redactCommandLogRecord removes sensitive args, options, result, and error text', () => {
  const record = redactCommandLogRecord({
    id: 'log-1',
    timestamp: '2026-06-07T00:00:00.000Z',
    userId: 'u1',
    platform: 'onebot',
    command: 'debug',
    args: ['--token', 'tok_live_secret', '--note', 'safe'],
    options: {
      authorization: 'Bearer auth_secret',
      headers: {
        cookie: 'sid=cookie_secret',
      },
      nested: {
        apiKey: 'sk-live-secret',
      },
    },
    success: false,
    error: 'Authorization: Bearer error_secret',
    executionTime: 1,
    result: 'callback?access_token=result_secret',
    isPrivate: true,
  })
  const payload = JSON.stringify(record)

  for (const secret of [
    'tok_live_secret',
    'auth_secret',
    'cookie_secret',
    'sk-live-secret',
    'error_secret',
    'result_secret',
  ]) {
    assert.doesNotMatch(payload, new RegExp(secret))
  }
  assert.match(payload, /\[REDACTED\]/)
  assert.deepEqual(record.args, ['--token', '[REDACTED]', '--note', 'safe'])
})

test('redactSensitiveValue handles circular objects without exposing sensitive fields', () => {
  const value: Record<string, unknown> = {
    token: 'cycle_secret',
    child: {
      apiKey: 'sk-cycle-secret',
    },
  }
  value.self = value

  const redacted = redactSensitiveValue(value)
  const payload = JSON.stringify(redacted)

  assert.doesNotMatch(payload, /cycle_secret/)
  assert.doesNotMatch(payload, /sk-cycle-secret/)
  assert.match(payload, /\[Circular\]/)
})

test('redactSensitiveText masks common header, query, and OpenAI key forms', () => {
  const text = redactSensitiveText(
    'Authorization: Bearer auth_secret Cookie: sid=cookie_secret url=https://a.test/callback?token=query_secret apiKey=sk-live-secret authorization=Basic basic_secret x-api-key=header_secret',
  )

  assert.doesNotMatch(text, /auth_secret/)
  assert.doesNotMatch(text, /cookie_secret/)
  assert.doesNotMatch(text, /query_secret/)
  assert.doesNotMatch(text, /sk-live-secret/)
  assert.doesNotMatch(text, /basic_secret/)
  assert.doesNotMatch(text, /header_secret/)
})
