import assert from 'node:assert/strict'
import test from 'node:test'

import {
  DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
  GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE,
  GroupGuardBehaviorSettingsStore,
} from './behavior-settings'

test('GroupGuardBehaviorSettingsStore creates default runtime settings', async () => {
  const created: unknown[] = []
  const store = new GroupGuardBehaviorSettingsStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return []
      },
      async create(table: string, record: unknown) {
        assert.equal(table, GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE)
        created.push(record)
      },
    },
  } as never)

  const settings = await store.getSettings()

  assert.equal(settings.id, 'default')
  assert.deepEqual(settings.fun, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.fun)
  assert.deepEqual(settings.moderation, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation)
  assert.ok(settings.createdAt instanceof Date)
  assert.ok(settings.updatedAt instanceof Date)
  assert.equal(created.length, 1)
})

test('GroupGuardBehaviorSettingsStore saves partial settings without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-09T10:00:00.000Z')
  const store = new GroupGuardBehaviorSettingsStore({
    database: {
      async get() {
        return [{
          id: 'default',
          fun: DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.fun,
          moderation: DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation,
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never)

  const saved = await store.saveSettings({
    fun: {
      diceSides: 20,
      muteLotteryPityThreshold: 3,
    },
    moderation: {
      warningThresholdExpression: ' warnings >= 1 ',
      defaultMuteSeconds: 180,
      antiRecallNotify: false,
    },
  })

  assert.equal(saved.id, 'default')
  assert.equal(saved.fun.diceSides, 20)
  assert.equal(saved.fun.muteLotteryBaseSeconds, 120)
  assert.equal(saved.fun.muteLotteryPityThreshold, 3)
  assert.equal(saved.moderation.warningThresholdExpression, 'warnings >= 1')
  assert.equal(saved.moderation.defaultMuteSeconds, 180)
  assert.equal(saved.moderation.antiRecallNotify, false)
  assert.equal(saved.moderation.repeatThreshold, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation.repeatThreshold)
  assert.equal(saved.createdAt, now)
  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.deepEqual(writes[0].fun, saved.fun)
  assert.deepEqual(writes[0].moderation, saved.moderation)
  assert.ok(writes[0].updatedAt instanceof Date)
})

test('GroupGuardBehaviorSettingsStore normalizes old records with missing fields', async () => {
  const store = new GroupGuardBehaviorSettingsStore({
    database: {
      async get() {
        return [{
          id: 'default',
          fun: {
            diceSides: 12,
          },
          moderation: {
            repeatThreshold: 4,
          },
          createdAt: new Date('2026-06-09T10:00:00.000Z'),
          updatedAt: new Date('2026-06-09T10:00:00.000Z'),
        }]
      },
    },
  } as never)

  const settings = await store.getSettings()

  assert.equal(settings.fun.diceSides, 12)
  assert.equal(settings.fun.muteLotteryBaseSeconds, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.fun.muteLotteryBaseSeconds)
  assert.equal(settings.fun.muteLotteryPitySeconds, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.fun.muteLotteryPitySeconds)
  assert.equal(settings.moderation.repeatThreshold, 4)
  assert.equal(settings.moderation.repeatWindowSize, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation.repeatWindowSize)
  assert.equal(settings.moderation.warningThresholdExpression, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation.warningThresholdExpression)
  assert.equal(settings.moderation.defaultMuteSeconds, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation.defaultMuteSeconds)
  assert.equal(settings.moderation.antiRecallNotify, DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS.moderation.antiRecallNotify)
})
