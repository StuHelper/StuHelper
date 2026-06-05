import assert from 'node:assert/strict'
import test from 'node:test'

import {
  GUARD_GROUP_BINDING_TABLE,
  GUARD_TEMPLATE_TABLE,
} from './policy'
import { GuardPolicyStore } from './policy-store'

test('GuardPolicyStore saveTemplate updates mutable fields without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-05T03:00:00.000Z')
  const store = new GuardPolicyStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, GUARD_TEMPLATE_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return [{
          id: 'default',
          name: '旧模板',
          muteDurationSeconds: 60,
          kickAfterMinutes: 10,
          reminderTemplate: 'old',
          exemptUsers: [],
          enabled: true,
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, GUARD_TEMPLATE_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never)

  await store.saveTemplate({
    id: 'default',
    name: '新模板',
    muteDurationSeconds: 120,
    kickAfterMinutes: 20,
    reminderTemplate: 'new',
    exemptUsers: ['10001'],
    enabled: false,
  })

  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.equal(writes[0].name, '新模板')
  assert.equal(writes[0].muteDurationSeconds, 120)
  assert.equal(writes[0].kickAfterMinutes, 20)
  assert.equal(writes[0].reminderTemplate, 'new')
  assert.deepEqual(writes[0].exemptUsers, ['10001'])
  assert.equal(writes[0].enabled, false)
  assert.ok(writes[0].updatedAt instanceof Date)
})

test('GuardPolicyStore saveBinding updates mutable fields without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-05T03:00:00.000Z')
  const store = new GuardPolicyStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, GUARD_GROUP_BINDING_TABLE)
        assert.deepEqual(query, { id: 'qq:10001' })
        return [{
          id: 'qq:10001',
          platform: 'qq',
          guildId: '10001',
          templateId: 'default',
          enabled: true,
          note: null,
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, GUARD_GROUP_BINDING_TABLE)
        assert.deepEqual(query, { id: 'qq:10001' })
        writes.push(patch)
      },
    },
  } as never)

  await store.saveBinding({
    platform: 'qq',
    guildId: '10001',
    templateId: 'strict',
    enabled: false,
    note: '',
  })

  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.equal(writes[0].platform, 'qq')
  assert.equal(writes[0].guildId, '10001')
  assert.equal(writes[0].templateId, 'strict')
  assert.equal(writes[0].enabled, false)
  assert.equal(writes[0].note, null)
  assert.ok(writes[0].updatedAt instanceof Date)
})
