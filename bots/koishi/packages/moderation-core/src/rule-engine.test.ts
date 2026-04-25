import assert from 'node:assert/strict'
import test from 'node:test'

import {
  detectRepeatTrigger,
  evaluateThresholdExpression,
  matchKeywordRules,
  normalizeModerationContent,
  type KeywordRuleRecord,
} from './rule-engine.ts'

test('阈值表达式支持比较与逻辑运算', () => {
  assert.equal(
    evaluateThresholdExpression('warnings >= 3 && repeats >= 2', {
      warnings: 3,
      repeats: 2,
      reports: 0,
    }),
    true,
  )

  assert.equal(
    evaluateThresholdExpression('warnings >= 4 || reports >= 1', {
      warnings: 3,
      repeats: 0,
      reports: 1,
    }),
    true,
  )

  assert.equal(
    evaluateThresholdExpression('warnings >= 4 && reports >= 1', {
      warnings: 3,
      repeats: 0,
      reports: 1,
    }),
    false,
  )
})

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
