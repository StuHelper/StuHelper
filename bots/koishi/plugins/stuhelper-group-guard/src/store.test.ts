import assert from 'node:assert/strict'
import test from 'node:test'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'

import { GuardMemberStore } from './store'

test('GuardMemberStore pushes active filters into database queries', async () => {
  const queries: unknown[] = []
  const store = new GuardMemberStore({
    database: {
      async get(_table: string, query: unknown) {
        queries.push(query)
        return []
      },
    },
  } as any)

  await store.listActive()
  await store.listPendingByGuild('guild-1')
  await store.findActiveByAdmissionSessionID('session-1')

  assert.deepEqual(queries, [
    { releasedAt: null, kickedAt: null },
    { guildId: 'guild-1', releasedAt: null, kickedAt: null },
    { admissionSessionID: 'session-1', releasedAt: null, kickedAt: null },
  ])
})

test('GuardMemberStore marks active records through active-only queries', async () => {
  const now = new Date('2026-05-05T12:30:00Z')
  const writes: Array<{
    query: Record<string, unknown>
    patch: Record<string, unknown>
  }> = []
  const store = new GuardMemberStore({
    database: {
      async set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        writes.push({ query, patch })
        return { matched: 1, modified: 1 }
      },
    },
  } as any)

  assert.equal(await store.markMuted('gm-1', now), true)
  assert.equal(await store.markReminderSent('gm-1', now), true)
  assert.equal(await store.markReleased('gm-1', now), true)
  assert.equal(await store.markKicked('gm-1', now), true)
  assert.equal(await store.markLastError('gm-1', 'send failed', now), true)
  assert.equal(await store.markBackendSynced('gm-1', {
    admissionSessionID: 'session-1',
    backendSyncPending: false,
    deadlineAt: now,
    nextReminderAt: now,
    manualReviewDeadlineAt: null,
  }), true)

  for (const write of writes) {
    assert.deepEqual(write.query, { id: 'gm-1', releasedAt: null, kickedAt: null })
    assert.equal('id' in write.patch, false)
    assert.equal('platform' in write.patch, false)
    assert.equal('botSelfId' in write.patch, false)
    assert.equal('guildId' in write.patch, false)
    assert.equal('memberId' in write.patch, false)
  }
  assert.deepEqual(writes.map((write) => write.patch), [
    { mutedAt: now, updatedAt: now },
    { reminderSentAt: now, updatedAt: now },
    { releasedAt: now, lastError: null, updatedAt: now },
    { kickedAt: now, lastError: null, updatedAt: now },
    { lastError: 'send failed', updatedAt: now },
    {
      admissionSessionID: 'session-1',
      backendSyncPending: false,
      deadlineAt: now,
      nextReminderAt: now,
      manualReviewDeadlineAt: null,
      lastError: null,
      updatedAt: writes[5].patch.updatedAt,
    },
  ])
  assert.ok(writes[5].patch.updatedAt instanceof Date)
})

test('GuardMemberStore savePending preserves the previous lastError on re-entry', async () => {
  const existing = guardRecord({ lastError: 'previous kick failed' })
  const patches: Array<Record<string, unknown>> = []
  const store = new GuardMemberStore({
    database: {
      async get(table: string, query: { id?: string }) {
        assert.equal(table, GUARD_MEMBER_TABLE)
        return query.id === existing.id ? [existing] : []
      },
      async set(_table: string, _query: unknown, patch: Record<string, unknown>) {
        patches.push(patch)
      },
    },
  } as any)

  await store.savePending(guardRecord({
    admissionSessionID: 'session-new',
    lastError: null,
  }))

  assert.equal(patches.length, 1)
  assert.equal('platform' in patches[0], false)
  assert.equal('botSelfId' in patches[0], false)
  assert.equal('guildId' in patches[0], false)
  assert.equal('memberId' in patches[0], false)
  assert.equal(patches[0].lastError, 'previous kick failed')
})

function guardRecord(overrides: Record<string, unknown> = {}) {
  const now = new Date('2026-05-05T12:00:00Z')
  return {
    id: 'mock:514:guild-1:10001',
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    memberId: '10001',
    memberName: 'Alice',
    verificationState: 'bound_unverified',
    admissionSessionID: 'session-old',
    backendSyncPending: false,
    joinedAt: now,
    deadlineAt: now,
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: now,
    reminderSentAt: now,
    releasedAt: null,
    kickedAt: now,
    lastError: null,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  }
}
