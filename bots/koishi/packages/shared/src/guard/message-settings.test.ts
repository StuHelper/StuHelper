import assert from 'node:assert/strict'
import test from 'node:test'

import {
  GROUP_GUARD_MESSAGE_KEYS,
  GROUP_GUARD_MESSAGE_SETTINGS_TABLE,
  GroupGuardMessageSettingsStore,
} from './message-settings'

test('GroupGuardMessageSettingsStore saveSettings updates message fields without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-09T10:00:00.000Z')
  const store = new GroupGuardMessageSettingsStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, GROUP_GUARD_MESSAGE_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return [{
          id: 'default',
          messages: {
            admissionReminder: '旧提醒 {authURL}',
          },
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, GROUP_GUARD_MESSAGE_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never)

  const saved = await store.saveSettings({
    messages: {
      admissionReminder: '新提醒 {authURL}',
      commandAccessDenied: '自定义：无权限。',
    },
  })

  assert.equal(saved.id, 'default')
  assert.equal(saved.messages.admissionReminder, '新提醒 {authURL}')
  assert.equal(saved.messages.commandAccessDenied, '自定义：无权限。')
  assert.equal(saved.messages.publicCommandsDisabled, '公开命令已由 StuHelper WebUI 关闭。')
  assert.equal(saved.createdAt, now)
  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.ok(writes[0].updatedAt instanceof Date)
  assert.equal((writes[0].messages as Record<string, string>).admissionReminder, '新提醒 {authURL}')
})

test('GROUP_GUARD_MESSAGE_KEYS is derived from default message templates', () => {
  assert.ok(GROUP_GUARD_MESSAGE_KEYS.includes('admissionReminder'))
  assert.ok(GROUP_GUARD_MESSAGE_KEYS.includes('reportHighRisk'))
  assert.ok(GROUP_GUARD_MESSAGE_KEYS.includes('moderationRepeatAutoMuteReason'))
})
