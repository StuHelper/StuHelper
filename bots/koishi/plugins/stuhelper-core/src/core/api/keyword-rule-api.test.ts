import assert from 'node:assert/strict'
import test from 'node:test'

import {
  parseKeywordRuleInput,
  toPublicKeywordRule,
} from './keyword-rule-api'

test('parseKeywordRuleInput accepts a scoped keyword rule', () => {
  const input = parseKeywordRuleInput({
    id: 'rule.qq:178037297',
    guildId: ' 178037297 ',
    pattern: ' 广告 ',
    matchMode: 'includes',
    action: 'delete',
    enabled: true,
    muteSeconds: 0,
    note: ' 违禁广告 ',
  })

  assert.deepEqual(input, {
    id: 'rule.qq:178037297',
    guildId: '178037297',
    pattern: '广告',
    matchMode: 'includes',
    action: 'delete',
    enabled: true,
    muteSeconds: 0,
    note: '违禁广告',
  })
})

test('parseKeywordRuleInput accepts safe regex rules and nullable notes', () => {
  const input = parseKeywordRuleInput({
    id: 'rule-global',
    guildId: '*',
    pattern: '^spam\\d{1,3}$',
    matchMode: 'regex',
    action: 'mute',
    enabled: false,
    muteSeconds: 600,
    note: '',
  })

  assert.deepEqual(input, {
    id: 'rule-global',
    guildId: '*',
    pattern: '^spam\\d{1,3}$',
    matchMode: 'regex',
    action: 'mute',
    enabled: false,
    muteSeconds: 600,
    note: null,
  })
})

test('parseKeywordRuleInput rejects unknown fields and invalid native values', () => {
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'rule-1',
      guildId: '*',
      pattern: 'spam',
      matchMode: 'includes',
      action: 'warn',
      enabled: true,
      muteSeconds: 0,
      note: null,
      platform: 'qq',
    }),
    /unsupported field: platform/,
  )
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'bad id',
      guildId: '*',
      pattern: 'spam',
      matchMode: 'includes',
      action: 'warn',
      enabled: true,
      muteSeconds: 0,
      note: null,
    }),
    /id must only contain/,
  )
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'rule-1',
      guildId: 'group-1',
      pattern: 'spam',
      matchMode: 'includes',
      action: 'warn',
      enabled: true,
      muteSeconds: 0,
      note: null,
    }),
    /guildId must be a numeric group id or \*/,
  )
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'rule-1',
      guildId: '*',
      pattern: 'spam',
      matchMode: 'contains',
      action: 'warn',
      enabled: true,
      muteSeconds: 0,
      note: null,
    }),
    /matchMode must be includes or regex/,
  )
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'rule-1',
      guildId: '*',
      pattern: 'spam',
      matchMode: 'includes',
      action: 'block',
      enabled: true,
      muteSeconds: 0,
      note: null,
    }),
    /action must be warn, delete, mute or review/,
  )
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'rule-1',
      guildId: '*',
      pattern: 'spam',
      matchMode: 'includes',
      action: 'mute',
      enabled: true,
      muteSeconds: 2_592_001,
      note: null,
    }),
    /muteSeconds must be at most 2592000/,
  )
})

test('parseKeywordRuleInput rejects unsafe regular expressions', () => {
  assert.throws(
    () => parseKeywordRuleInput({
      id: 'rule-1',
      guildId: '*',
      pattern: '(a+)+$',
      matchMode: 'regex',
      action: 'warn',
      enabled: true,
      muteSeconds: 0,
      note: null,
    }),
  )
})

test('toPublicKeywordRule serializes dates without leaking internal fields', () => {
  const data = toPublicKeywordRule({
    id: 'rule-1',
    guildId: '*',
    pattern: 'spam',
    matchMode: 'includes',
    action: 'warn',
    enabled: true,
    muteSeconds: 0,
    note: null,
    createdAt: new Date('2026-06-09T10:00:00.000Z'),
    updatedAt: new Date('2026-06-09T10:01:00.000Z'),
  })

  assert.deepEqual(data, {
    id: 'rule-1',
    guildId: '*',
    pattern: 'spam',
    matchMode: 'includes',
    action: 'warn',
    enabled: true,
    muteSeconds: 0,
    note: null,
    createdAt: '2026-06-09T10:00:00.000Z',
    updatedAt: '2026-06-09T10:01:00.000Z',
  })
})
