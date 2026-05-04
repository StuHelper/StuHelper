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
  await store.findActiveByAdmissionSessionID('session-1')

  assert.deepEqual(queries, [
    { releasedAt: null, kickedAt: null },
    { admissionSessionID: 'session-1', releasedAt: null, kickedAt: null },
  ])
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
