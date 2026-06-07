import assert from 'node:assert/strict'
import test from 'node:test'

import { EntityPageService, type EntityPageServiceDeps } from './entity-page.service'

const SCOPE_1001 = { guildIds: new Set(['1001']) }

test('EntityPageService filters user profile facts to the active guild scope', async () => {
  const service = createEntityPageService()

  const profile = await service.getProfile({ kind: 'user', id: 'target' }, SCOPE_1001)

  assert.equal(profile.kind, 'user')
  assert.deepEqual(profile.warns.map((item) => item.guildId), ['1001'])
  assert.deepEqual(profile.restricted.map((item) => item.guildId), ['1001'])
  assert.deepEqual(profile.reviews.map((item) => item.guildId), ['1001'])
  assert.deepEqual(profile.reports.map((item) => item.guildId), ['1001'])
  assert.deepEqual(profile.recentEvents.map((item) => item.guildId), ['1001'])
  assert.equal(profile.blacklist, null)
  assert.equal(profile.summary.totalWarns, 2)
  assert.equal(profile.summary.pendingReviews, 1)
  assert.equal(profile.summary.openReports, 1)
})

test('EntityPageService rejects out-of-scope guild profile requests', async () => {
  const service = createEntityPageService()

  await assert.rejects(
    () => service.getProfile({ kind: 'guild', id: '2002' }, SCOPE_1001),
    /outside of the current console guild scope/,
  )
})

test('EntityPageService keeps global profile requests unfiltered', async () => {
  const service = createEntityPageService()

  const profile = await service.getProfile({ kind: 'user', id: 'target' })

  assert.equal(profile.kind, 'user')
  assert.deepEqual(profile.warns.map((item) => item.guildId), ['2002', '1001'])
  assert.equal(profile.blacklist?.userId, 'target')
  assert.equal(profile.summary.totalWarns, 5)
})

test('EntityPageService reports admission policy separately from group config', async () => {
  const service = createEntityPageService()

  const staticAdmission = await service.getProfile({ kind: 'guild', id: '2002' })
  const configuredAdmission = await service.getProfile({ kind: 'guild', id: '1001' })

  assert.equal(staticAdmission.kind, 'guild')
  assert.equal(staticAdmission.summary.configured, false)
  assert.equal(staticAdmission.summary.admissionConfigured, true)

  assert.equal(configuredAdmission.kind, 'guild')
  assert.equal(configuredAdmission.summary.configured, true)
  assert.equal(configuredAdmission.summary.admissionConfigured, false)
})

function createEntityPageService() {
  return new EntityPageService(createDeps())
}

function createDeps(): EntityPageServiceDeps {
  return {
    loadWarns: async () => ({
      '1001': { target: { count: 2, timestamp: 1 } },
      '2002': { target: { count: 3, timestamp: 1 } },
    }),
    loadBlacklist: async () => ({
      target: { userId: 'target', guildId: '2002', timestamp: 1 },
    }),
    loadGuardRecords: async () => [
      createGuardRecord('gm-1001', '1001'),
      createGuardRecord('gm-2002', '2002'),
    ],
    loadReviews: async () => [
      createReview('rv-1001', '1001'),
      createReview('rv-2002', '2002'),
    ],
    loadReports: async () => [
      createReport('rp-1001', '1001'),
      createReport('rp-2002', '2002'),
    ],
    loadEvents: async () => [
      createEvent('ev-1001', '1001'),
      createEvent('ev-2002', '2002'),
    ],
    hasGuildConfig: async (guildId) => guildId === '1001',
    hasAdmissionPolicy: async (guildId) => guildId === '2002',
    resolveGuildName: (guildId) => ({ name: `Guild ${guildId}`, avatar: null }),
    resolveUserName: (userId) => ({ name: `User ${userId}`, avatar: null }),
  }
}

function createGuardRecord(id: string, guildId: string) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    memberId: 'target',
    memberName: 'Target',
    verificationState: 'bound_unverified',
    joinedAt: new Date('2026-04-23T07:30:00.000Z'),
    deadlineAt: new Date('2026-04-23T08:30:00.000Z'),
    mutedAt: new Date('2026-04-23T07:31:00.000Z'),
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: new Date('2026-04-23T07:30:00.000Z'),
    updatedAt: new Date('2026-04-23T07:31:00.000Z'),
  }
}

function createReview(id: string, guildId: string) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    memberId: 'target',
    actionType: 'kick',
    status: 'pending',
    reason: 'review reason',
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt: new Date('2026-04-23T07:40:00.000Z'),
    updatedAt: new Date('2026-04-23T07:40:00.000Z'),
  }
}

function createReport(id: string, guildId: string) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    reporterMemberId: 'reporter',
    targetMemberId: 'target',
    reason: 'report reason',
    aiStatus: 'pending',
    aiSeverity: 'medium',
    aiSummary: null,
    createdAt: new Date('2026-04-23T07:45:00.000Z'),
    updatedAt: new Date('2026-04-23T07:45:00.000Z'),
  }
}

function createEvent(id: string, guildId: string) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    memberId: 'target',
    type: 'review_created',
    level: 'high',
    summary: 'event summary',
    payload: { reviewId: id },
    createdAt: new Date('2026-04-23T07:50:00.000Z'),
    updatedAt: new Date('2026-04-23T07:50:00.000Z'),
  }
}
