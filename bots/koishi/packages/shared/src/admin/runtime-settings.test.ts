import assert from 'node:assert/strict'
import test from 'node:test'

import {
  ADMIN_RUNTIME_SETTINGS_TABLE,
  AdminRuntimeSettingsStore,
} from './runtime-settings'

test('AdminRuntimeSettingsStore saveSettings updates message fields without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-09T08:30:00.000Z')
  const store = new AdminRuntimeSettingsStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, ADMIN_RUNTIME_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return [{
          id: 'default',
          messages: {
            guardPendingMembersHeader: '旧待认证：',
          },
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, ADMIN_RUNTIME_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never)

  const saved = await store.saveSettings({
    messages: {
      guardPendingMembersHeader: '新待认证：',
      commandAccessDenied: '自定义：无权限。',
    },
  })

  assert.equal(saved.id, 'default')
  assert.equal(saved.messages.guardPendingMembersHeader, '新待认证：')
  assert.equal(saved.messages.commandAccessDenied, '自定义：无权限。')
  assert.equal(saved.messages.guardPendingMemberLine, '{memberId} 截止 {deadlineAt}')
  assert.equal(saved.createdAt, now)
  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.ok(writes[0].updatedAt instanceof Date)
  assert.equal((writes[0].messages as Record<string, string>).guardPendingMembersHeader, '新待认证：')
})
