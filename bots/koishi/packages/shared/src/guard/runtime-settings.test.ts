import assert from 'node:assert/strict'
import test from 'node:test'

import {
  ADMISSION_RUNTIME_SETTINGS_TABLE,
  AdmissionRuntimeSettingsStore,
} from './runtime-settings'

test('AdmissionRuntimeSettingsStore saveSettings updates mutable fields without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-05T02:00:00.000Z')
  const store = new AdmissionRuntimeSettingsStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, ADMISSION_RUNTIME_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return [{
          id: 'default',
          actionStreamEnabled: true,
          publicCommandsEnabled: false,
          admissionCommandsEnabled: true,
          moderationEnabled: false,
          freshmanForwardEnabled: false,
          fallbackScanEnabled: true,
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, ADMISSION_RUNTIME_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never, {
    actionStreamEnabled: true,
    publicCommandsEnabled: false,
    admissionCommandsEnabled: true,
    moderationEnabled: false,
    freshmanForwardEnabled: false,
    fallbackScanEnabled: true,
  })

  const saved = await store.saveSettings({
    actionStreamEnabled: false,
    moderationEnabled: true,
  })

  assert.equal(saved.id, 'default')
  assert.equal(saved.actionStreamEnabled, false)
  assert.equal(saved.moderationEnabled, true)
  assert.equal(saved.createdAt, now)
  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.equal(writes[0].actionStreamEnabled, false)
  assert.equal(writes[0].moderationEnabled, true)
  assert.ok(writes[0].updatedAt instanceof Date)
})
