import assert from 'node:assert/strict'
import test from 'node:test'

import {
  detectRepeatTrigger,
  matchKeywordRules,
  normalizeModerationContent,
  type KeywordRuleRecord,
} from './rule-engine.ts'

test('复读检测在达到阈值时命中', () => {
  const result = detectRepeatTrigger([
    { normalizedContent: '航小伴', memberId: '10001' },
    { normalizedContent: '航小伴', memberId: '10002' },
  ], '航小伴', 3)

  assert.equal(result.hit, true)
  assert.equal(result.count, 3)
})

test('关键词规则会生成动作列表', () => {
  const rules: KeywordRuleRecord[] = [
    {
      id: 'rule-delete',
      guildId: 'group-1',
      pattern: '广告',
      matchMode: 'includes',
      action: 'delete',
      enabled: true,
      muteSeconds: 0,
      note: '广告词',
      createdAt: new Date(),
      updatedAt: new Date(),
    },
    {
      id: 'rule-mute',
      guildId: 'group-1',
      pattern: '^http',
      matchMode: 'regex',
      action: 'mute',
      enabled: true,
      muteSeconds: 600,
      note: '链接拦截',
      createdAt: new Date(),
      updatedAt: new Date(),
    },
  ]

  const hits = matchKeywordRules(rules, {
    guildId: 'group-1',
    content: 'http://example.com 广告',
    normalizedContent: normalizeModerationContent('http://example.com 广告'),
  })

  assert.equal(hits.length, 2)
  assert.deepEqual(hits.map((item) => item.action), ['delete', 'mute'])
  assert.equal(hits[1].muteSeconds, 600)
})

test('非法正则关键词规则被逐条跳过并上报失败，不中断匹配', () => {
  const rules: KeywordRuleRecord[] = [
    {
      id: 'unsafe-regex',
      guildId: 'group-1',
      pattern: '(a+)+$',
      matchMode: 'regex',
      action: 'mute',
      enabled: true,
      muteSeconds: 600,
      note: 'unsafe',
      createdAt: new Date(),
      updatedAt: new Date(),
    },
    {
      id: 'safe-includes',
      guildId: 'group-1',
      pattern: 'aaaa',
      matchMode: 'includes',
      action: 'delete',
      enabled: true,
      muteSeconds: 0,
      note: 'safe',
      createdAt: new Date(),
      updatedAt: new Date(),
    },
  ]
  const failures: Array<{ ruleId: string; message: string }> = []

  const hits = matchKeywordRules(rules, {
    guildId: 'group-1',
    content: 'aaaaaaaaaaaaaaaa!',
    normalizedContent: normalizeModerationContent('aaaaaaaaaaaaaaaa!'),
  }, ({ rule, error }) => {
    failures.push({
      ruleId: rule.id,
      message: error instanceof Error ? error.message : String(error),
    })
  })

  assert.deepEqual(hits.map((item) => item.ruleId), ['safe-includes'])
  assert.equal(failures.length, 1)
  assert.equal(failures[0].ruleId, 'unsafe-regex')
  assert.match(failures[0].message, /unsafe keyword regex/)
})

test('非法正则关键词规则在未提供失败回调时也不抛错', () => {
  const rules: KeywordRuleRecord[] = [
    {
      id: 'unsafe-regex',
      guildId: 'group-1',
      pattern: '(a+)+$',
      matchMode: 'regex',
      action: 'mute',
      enabled: true,
      muteSeconds: 600,
      note: 'unsafe',
      createdAt: new Date(),
      updatedAt: new Date(),
    },
  ]

  const hits = matchKeywordRules(rules, {
    guildId: 'group-1',
    content: 'aaaaaaaaaaaaaaaa!',
    normalizedContent: normalizeModerationContent('aaaaaaaaaaaaaaaa!'),
  })

  assert.deepEqual(hits, [])
})
