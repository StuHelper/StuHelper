import assert from 'node:assert/strict'
import test from 'node:test'

import {
  DEFAULT_GROUP_GUARD_AI_SETTINGS,
  GROUP_GUARD_AI_SETTINGS_TABLE,
  GroupGuardAISettingsStore,
} from './ai-settings'

test('GroupGuardAISettingsStore creates default runtime settings', async () => {
  const created: unknown[] = []
  const store = new GroupGuardAISettingsStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, GROUP_GUARD_AI_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return []
      },
      async create(table: string, record: unknown) {
        assert.equal(table, GROUP_GUARD_AI_SETTINGS_TABLE)
        created.push(record)
      },
    },
  } as never)

  const settings = await store.getSettings()

  assert.equal(settings.id, 'default')
  assert.equal(settings.enabled, DEFAULT_GROUP_GUARD_AI_SETTINGS.enabled)
  assert.equal(settings.endpoint, DEFAULT_GROUP_GUARD_AI_SETTINGS.endpoint)
  assert.equal(settings.apiKey, DEFAULT_GROUP_GUARD_AI_SETTINGS.apiKey)
  assert.equal(settings.model, DEFAULT_GROUP_GUARD_AI_SETTINGS.model)
  assert.ok(settings.createdAt instanceof Date)
  assert.ok(settings.updatedAt instanceof Date)
  assert.equal(created.length, 1)
})

test('GroupGuardAISettingsStore saves partial settings and preserves apiKey when omitted', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-09T10:00:00.000Z')
  const store = new GroupGuardAISettingsStore({
    database: {
      async get() {
        return [{
          id: 'default',
          enabled: false,
          endpoint: 'https://old.example.test/review',
          apiKey: 'old-key',
          model: 'old-model',
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, GROUP_GUARD_AI_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never)

  const saved = await store.saveSettings({
    enabled: true,
    endpoint: ' https://new.example.test/review ',
    model: ' gpt-test ',
  })

  assert.equal(saved.id, 'default')
  assert.equal(saved.enabled, true)
  assert.equal(saved.endpoint, 'https://new.example.test/review')
  assert.equal(saved.apiKey, 'old-key')
  assert.equal(saved.model, 'gpt-test')
  assert.equal(saved.createdAt, now)
  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.equal('apiKey' in writes[0], false)
  assert.ok(writes[0].updatedAt instanceof Date)
})

test('GroupGuardAISettingsStore replaces and clears apiKey explicitly', async () => {
  const writes: Array<Record<string, unknown>> = []
  const store = new GroupGuardAISettingsStore({
    database: {
      async get() {
        return [{
          id: 'default',
          enabled: true,
          endpoint: 'https://example.test/review',
          apiKey: 'old-key',
          model: 'gpt-test',
          createdAt: new Date('2026-06-09T10:00:00.000Z'),
          updatedAt: new Date('2026-06-09T10:00:00.000Z'),
        }]
      },
      async set(_table: string, _query: unknown, patch: Record<string, unknown>) {
        writes.push(patch)
      },
    },
  } as never)

  const replaced = await store.saveSettings({ apiKey: ' new-key ' })
  const cleared = await store.saveSettings({ apiKey: '   ' })

  assert.equal(replaced.apiKey, 'new-key')
  assert.equal(cleared.apiKey, '')
  assert.equal(writes[0].apiKey, 'new-key')
  assert.equal(writes[1].apiKey, '')
})

test('GroupGuardAISettingsStore normalizes old records with missing fields', async () => {
  const store = new GroupGuardAISettingsStore({
    database: {
      async get() {
        return [{
          id: 'default',
          enabled: true,
          endpoint: ' https://example.test/review ',
          createdAt: new Date('2026-06-09T10:00:00.000Z'),
          updatedAt: new Date('2026-06-09T10:00:00.000Z'),
        }]
      },
    },
  } as never)

  const settings = await store.getSettings()

  assert.equal(settings.enabled, true)
  assert.equal(settings.endpoint, 'https://example.test/review')
  assert.equal(settings.apiKey, DEFAULT_GROUP_GUARD_AI_SETTINGS.apiKey)
  assert.equal(settings.model, DEFAULT_GROUP_GUARD_AI_SETTINGS.model)
})
