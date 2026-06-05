import assert from 'node:assert/strict'
import test from 'node:test'

import { GUARD_MEMBER_TABLE } from '../types/index'
import { GuardMemberAdminStore } from './member-admin-store'

test('GuardMemberAdminStore lists active guild members through scoped database queries', async () => {
  const queries: Array<Record<string, unknown>> = []
  const store = new GuardMemberAdminStore({
    database: {
      async get(table: string, query: Record<string, unknown>) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        queries.push(query)
        return [
          createRecord('member-2', '2026-06-05T05:20:00.000Z'),
          createRecord('member-1', '2026-06-05T05:10:00.000Z'),
          createRecord('member-3', '2026-06-05T05:00:00.000Z'),
        ]
      },
    },
  } as never)

  const records = await store.listActiveByGuild({
    guildId: 'guild-1',
    memberIds: ['member-2', 'member-1'],
  })

  assert.deepEqual(queries, [{
    guildId: 'guild-1',
    releasedAt: null,
    kickedAt: null,
  }])
  assert.deepEqual(records.map((record) => record.memberId), ['member-1', 'member-2'])
})

test('GuardMemberAdminStore skips blank guild active lookups', async () => {
  const store = new GuardMemberAdminStore({
    database: {
      async get() {
        throw new Error('database should not be called')
      },
    },
  } as never)

  assert.deepEqual(await store.listActiveByGuild({ guildId: '  ' }), [])
})

test('GuardMemberAdminStore marks active guard members muted without identity churn', async () => {
  const updatedAt = new Date('2026-06-05T05:00:00.000Z')
  const mutedAt = new Date('2026-06-05T05:30:00.000Z')
  const writes: Array<{
    query: Record<string, unknown>
    patch: Record<string, unknown>
  }> = []
  const store = new GuardMemberAdminStore({
    database: {
      async set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        writes.push({ query, patch })
        return { matched: 1, modified: 1 }
      },
    },
  } as never)

  assert.equal(await store.tryMarkActiveMuted({
    record: { id: 'gm-1', updatedAt },
    mutedAt,
  }), true)

  assert.deepEqual(writes, [{
    query: { id: 'gm-1', updatedAt, releasedAt: null, kickedAt: null },
    patch: { mutedAt, updatedAt: mutedAt },
  }])
  assert.equal('id' in writes[0].patch, false)
  assert.equal('guildId' in writes[0].patch, false)
  assert.equal('memberId' in writes[0].patch, false)
})

function createRecord(memberId: string, deadlineAt: string) {
  const updatedAt = new Date('2026-06-05T05:00:00.000Z')
  return {
    id: `gm-${memberId}`,
    guildId: 'guild-1',
    channelId: 'guild-1',
    memberId,
    deadlineAt: new Date(deadlineAt),
    releasedAt: null,
    kickedAt: null,
    updatedAt,
  }
}
