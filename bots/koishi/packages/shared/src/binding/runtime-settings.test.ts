import assert from 'node:assert/strict'
import test from 'node:test'

import {
  BINDING_RUNTIME_SETTINGS_TABLE,
  BindingRuntimeSettingsStore,
} from './runtime-settings'

test('BindingRuntimeSettingsStore saveSettings updates command and message fields without primary key churn', async () => {
  const writes: Array<Record<string, unknown>> = []
  const now = new Date('2026-06-09T08:00:00.000Z')
  const store = new BindingRuntimeSettingsStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, BINDING_RUNTIME_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        return [{
          id: 'default',
          command: '绑定',
          messages: {
            directOnly: '旧私聊提示',
          },
          createdAt: now,
          updatedAt: now,
        }]
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        assert.equal(table, BINDING_RUNTIME_SETTINGS_TABLE)
        assert.deepEqual(query, { id: 'default' })
        writes.push(patch)
      },
    },
  } as never)

  const saved = await store.saveSettings({
    command: '  绑定账号  ',
    messages: {
      directOnly: '新私聊提示',
      missingCode: '请输入 {command} 后面的绑定码',
    },
  })

  assert.equal(saved.id, 'default')
  assert.equal(saved.command, '绑定账号')
  assert.equal(saved.messages.directOnly, '新私聊提示')
  assert.equal(saved.messages.missingCode, '请输入 {command} 后面的绑定码')
  assert.equal(saved.messages.successVerified, '绑定成功，当前账号已完成学生认证，加入受控群时会自动放行。')
  assert.equal(saved.createdAt, now)
  assert.equal(writes.length, 1)
  assert.equal('id' in writes[0], false)
  assert.equal('createdAt' in writes[0], false)
  assert.equal(writes[0].command, '绑定账号')
  assert.ok(writes[0].updatedAt instanceof Date)
  assert.equal((writes[0].messages as Record<string, string>).directOnly, '新私聊提示')
})

