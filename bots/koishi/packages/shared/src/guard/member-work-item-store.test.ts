import assert from 'node:assert/strict'
import test from 'node:test'

import { GUARD_MEMBER_TABLE } from '../types/index'
import { GuardMemberWorkItemStore } from './member-work-item-store'

test('GuardMemberWorkItemStore owns optimistic work item transitions', async () => {
  const updatedAt = new Date('2026-06-05T05:00:00.000Z')
  const claimedAt = new Date('2026-06-05T05:01:00.000Z')
  const releasedAt = new Date('2026-06-05T05:02:00.000Z')
  const kickedAt = new Date('2026-06-05T05:03:00.000Z')
  const deferredAt = new Date('2026-06-05T05:04:00.000Z')
  const deadlineAt = new Date('2026-06-05T05:34:00.000Z')
  const rolledBackAt = new Date('2026-06-05T05:05:00.000Z')
  const writes: Array<{
    query: Record<string, unknown>
    patch: Record<string, unknown>
  }> = []
  const store = new GuardMemberWorkItemStore({
    database: {
      async set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        writes.push({ query, patch })
        return { matched: 1, modified: 1 }
      },
    },
  } as never)

  const record = { id: 'gm-1', updatedAt }
  assert.equal(await store.tryClaimActive({ record, claimedAt }), true)
  assert.equal(await store.tryReleaseClaimed({ guardId: record.id, claimedAt, releasedAt }), true)
  assert.equal(await store.tryKickClaimed({ guardId: record.id, claimedAt, kickedAt }), true)
  assert.equal(await store.tryDeferActive({ record, deadlineAt, updatedAt: deferredAt }), true)
  await store.rollbackClaim({
    guardId: record.id,
    claimedAt,
    rolledBackAt,
    error: new Error('send failed'),
  })

  assert.deepEqual(writes, [
    {
      query: { id: 'gm-1', updatedAt, releasedAt: null, kickedAt: null },
      patch: { updatedAt: claimedAt },
    },
    {
      query: { id: 'gm-1', updatedAt: claimedAt, releasedAt: null, kickedAt: null },
      patch: { releasedAt, lastError: null, updatedAt: releasedAt },
    },
    {
      query: { id: 'gm-1', updatedAt: claimedAt, releasedAt: null, kickedAt: null },
      patch: { kickedAt, lastError: null, updatedAt: kickedAt },
    },
    {
      query: { id: 'gm-1', updatedAt, releasedAt: null, kickedAt: null },
      patch: { deadlineAt, lastError: null, updatedAt: deferredAt },
    },
    {
      query: { id: 'gm-1', updatedAt: claimedAt, releasedAt: null, kickedAt: null },
      patch: { lastError: 'send failed', updatedAt: rolledBackAt },
    },
  ])

  for (const write of writes) {
    assert.equal('id' in write.patch, false)
    assert.equal('platform' in write.patch, false)
    assert.equal('botSelfId' in write.patch, false)
    assert.equal('guildId' in write.patch, false)
    assert.equal('memberId' in write.patch, false)
  }
})

test('GuardMemberWorkItemStore reports lost active claims without throwing', async () => {
  const store = new GuardMemberWorkItemStore({
    database: {
      async set() {
        return { matched: 0, modified: 0 }
      },
    },
  } as never)

  assert.equal(await store.tryClaimActive({
    record: {
      id: 'gm-1',
      updatedAt: new Date('2026-06-05T05:00:00.000Z'),
    },
    claimedAt: new Date('2026-06-05T05:01:00.000Z'),
  }), false)
})

test('GuardMemberWorkItemStore marks active guard members kicked by member subject', async () => {
  const updatedAt = new Date('2026-06-05T05:00:00.000Z')
  const kickedAt = new Date('2026-06-05T05:06:00.000Z')
  const reads: Array<Record<string, unknown>> = []
  const writes: Array<{
    query: Record<string, unknown>
    patch: Record<string, unknown>
  }> = []
  const store = new GuardMemberWorkItemStore({
    database: {
      async get(table: string, query: Record<string, unknown>) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        reads.push(query)
        return [{
          id: 'gm-1',
          updatedAt,
          releasedAt: null,
          kickedAt: null,
        }]
      },
      async set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        writes.push({ query, patch })
        return { matched: 1, modified: 1 }
      },
    },
  } as never)

  assert.equal(await store.tryMarkActiveMemberKicked({
    botSelfId: 'bot',
    guildId: 'guild-1',
    memberId: 'member-1',
    kickedAt,
  }), true)

  assert.deepEqual(reads, [{
    botSelfId: 'bot',
    guildId: 'guild-1',
    memberId: 'member-1',
    releasedAt: null,
    kickedAt: null,
  }])
  assert.deepEqual(writes, [{
    query: { id: 'gm-1', updatedAt, releasedAt: null, kickedAt: null },
    patch: { kickedAt, lastError: null, updatedAt: kickedAt },
  }])
  assert.equal('id' in writes[0].patch, false)
  assert.equal('botSelfId' in writes[0].patch, false)
  assert.equal('guildId' in writes[0].patch, false)
  assert.equal('memberId' in writes[0].patch, false)
})
