import assert from 'node:assert/strict'
import test from 'node:test'

import {
  runSettingsSaveSteps,
  saveKeywordRuleChanges,
  SettingsSaveStepFailure,
  type SettingsSaveStepResult,
} from './settings-save'

test('runSettingsSaveSteps records confirmed, unconfirmed and not-run domains', async () => {
  const calls: string[] = []
  const snapshots: SettingsSaveStepResult[][] = []

  await assert.rejects(
    runSettingsSaveSteps([
      {
        key: 'core',
        label: '群管中心设置',
        run: async () => {
          calls.push('core')
        },
      },
      {
        key: 'binding',
        label: 'QQ 绑定提示',
        run: async () => {
          calls.push('binding')
          throw new Error('network timeout')
        },
      },
      {
        key: 'admin',
        label: '管理员命令提示',
        run: async () => {
          calls.push('admin')
        },
      },
    ], (results) => {
      snapshots.push(results.map((result) => ({ ...result })))
    }),
    (cause: unknown) => {
      assert.ok(cause instanceof SettingsSaveStepFailure)
      assert.equal(cause.stepKey, 'binding')
      assert.equal(cause.stepLabel, 'QQ 绑定提示')
      assert.match(cause.message, /network timeout/)
      return true
    },
  )

  assert.deepEqual(calls, ['core', 'binding'])
  assert.deepEqual(
    snapshots.at(-1)?.map(({ key, status }) => ({ key, status })),
    [
      { key: 'core', status: 'confirmed' },
      { key: 'binding', status: 'unconfirmed' },
      { key: 'admin', status: 'not-run' },
    ],
  )
})

test('saveKeywordRuleChanges advances the baseline after each confirmed mutation', async () => {
  interface Rule {
    id: string
    pattern: string
  }

  const compareRules = (left: Rule, right: Rule) => left.id.localeCompare(right.id)
  const baselineSnapshots: Rule[][] = []
  const calls: string[] = []
  const original = [
    { id: 'a', pattern: 'old-a' },
    { id: 'b', pattern: 'old-b' },
  ]
  const next = [
    { id: 'b', pattern: 'new-b' },
    { id: 'c', pattern: 'new-c' },
  ]

  await assert.rejects(
    saveKeywordRuleChanges({
      original,
      next,
      compareRules,
      deleteRule: async (id) => {
        calls.push(`delete:${id}`)
      },
      upsertRule: async (rule) => {
        calls.push(`upsert:${rule.id}`)
        if (rule.id === 'b') throw new Error('write rejected')
      },
      onBaselineChange: (rules) => {
        baselineSnapshots.push(structuredClone(rules) as Rule[])
      },
    }),
    /write rejected/,
  )

  assert.deepEqual(calls, ['delete:a', 'upsert:b'])
  assert.deepEqual(baselineSnapshots, [[{ id: 'b', pattern: 'old-b' }]])

  const retryCalls: string[] = []
  await saveKeywordRuleChanges({
    original: baselineSnapshots.at(-1) ?? [],
    next,
    compareRules,
    deleteRule: async (id) => {
      retryCalls.push(`delete:${id}`)
    },
    upsertRule: async (rule) => {
      retryCalls.push(`upsert:${rule.id}`)
    },
    onBaselineChange: () => undefined,
  })

  assert.deepEqual(retryCalls, ['upsert:b', 'upsert:c'])
})
