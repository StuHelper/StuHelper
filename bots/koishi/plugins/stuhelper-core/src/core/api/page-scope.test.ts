import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildScopedConfigGovernancePageData,
  buildScopedDashboardPageData,
  buildScopedIdentityPageData,
  buildScopedReviewPageData,
} from './page-scope'

const SCOPE_1001 = {
  kind: 'guilds' as const,
  guildIds: new Set(['1001']),
}

test('buildScopedDashboardPageData filters guild-bound dashboard collections', () => {
  const data = buildScopedDashboardPageData({
    generatedAt: '2026-04-23T08:00:00.000Z',
    pendingMembers: [
      createGuardMember('gm-1001', '1001'),
      createGuardMember('gm-2002', '2002'),
    ],
    pendingReviews: [
      createReview('rv-1001', '1001'),
      createReview('rv-2002', '2002'),
    ],
    recentEvents: [
      createEvent('ev-1001', '1001', 'high'),
      createEvent('ev-2002', '2002', 'critical'),
    ],
    recentReports: [
      createReport('rp-1001', '1001'),
      createReport('rp-2002', '2002'),
    ],
    commandPolicies: [createCommandPolicy('report')],
    guardTemplates: [createGuardTemplate('tpl-1')],
    guardBindings: [
      createGuardBinding('bind-1001', '1001'),
      createGuardBinding('bind-2002', '2002'),
    ],
  }, SCOPE_1001)

  assert.deepEqual(data.pendingMembers.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.pendingReviews.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.recentEvents.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.recentReports.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.guardBindings.map((item) => item.guildId), ['1001'])
  assert.equal(data.overview.pendingAdmissions, 1)
  assert.equal(data.overview.pendingReviews, 1)
  assert.equal(data.overview.openReports, 1)
  assert.equal(data.overview.highRiskEvents, 1)
})

test('buildScopedIdentityPageData drops members outside the active guild scope before recomputing summary', () => {
  const data = buildScopedIdentityPageData({
    generatedAt: '2026-04-23T08:00:00.000Z',
    guardRecords: [
      createGuardMember('gm-1001', '1001', { memberId: '2001' }),
      createGuardMember('gm-2002', '2002', { memberId: '2002' }),
      createGuardMember('gm-released', '1001', {
        memberId: '2003',
        releasedAt: new Date('2026-04-23T07:58:00.000Z'),
      }),
    ],
    verificationProfiles: [
      createProfile('2001', 'verified'),
      createProfile('2002', 'bound_unverified'),
      createProfile('2003', 'verified'),
    ],
    lookupErrors: [
      { memberId: '2001', message: 'transient error' },
      { memberId: '2002', message: 'out of scope' },
    ],
  }, SCOPE_1001)

  assert.deepEqual(data.members.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.groups.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.recentReleases.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.lookupErrors.map((item) => item.memberId), ['2001'])
  assert.equal(data.summary.pendingMembers, 1)
  assert.equal(data.summary.verifiedMembers, 2)
  assert.equal(data.summary.boundUnverifiedMembers, 0)
  assert.equal(data.summary.releasedMembers, 1)
})

test('buildScopedReviewPageData keeps only allowed guild work items and related events', () => {
  const data = buildScopedReviewPageData({
    generatedAt: '2026-04-23T08:00:00.000Z',
    pendingReviews: [
      createReview('rv-1001', '1001'),
      createReview('rv-2002', '2002'),
    ],
    pendingMembers: [
      createGuardMember('gm-1001', '1001'),
      createGuardMember('gm-2002', '2002'),
    ],
    reports: [
      createReport('rp-1001', '1001', { targetMemberId: '2001' }),
      createReport('rp-2002', '2002', { targetMemberId: '2002' }),
    ],
    events: [
      createEvent('ev-1001', '1001', 'medium', { reportId: 'rp-1001' }),
      createEvent('ev-2002', '2002', 'medium', { reportId: 'rp-2002' }),
    ],
  }, SCOPE_1001)

  assert.deepEqual(data.items.map((item) => item.guildId), ['1001', '1001', '1001'])
  assert.deepEqual(data.events.map((item) => item.guildId), ['1001'])
  assert.deepEqual(
    data.items.find((item) => item.kind === 'report')?.relatedEventIds,
    ['ev-1001'],
  )
})

test('buildScopedConfigGovernancePageData filters guild-bound governance records', () => {
  const data = buildScopedConfigGovernancePageData({
    generatedAt: '2026-04-23T08:00:00.000Z',
    groupConfigs: {
      '1001': { enabled: true },
      '2002': { enabled: false },
    },
    guildNames: {
      '1001': { name: '群 1001', avatar: 'avatar-1001' },
      '2002': { name: '群 2002', avatar: 'avatar-2002' },
    },
    templates: [createGuardTemplate('tpl-1')],
    bindings: [
      createGuardBinding('bind-1001', '1001'),
      createGuardBinding('bind-2002', '2002'),
    ],
    commandPolicies: [createCommandPolicy('report')],
    supportedCommandIds: ['report'],
  }, SCOPE_1001)

  assert.deepEqual(data.groupConfigs.map((item) => item.guildId), ['1001'])
  assert.deepEqual(data.bindings.map((item) => item.guildId), ['1001'])
  assert.equal(data.templates.length, 1)
  assert.equal(data.commandPolicies.length, 1)
})

function createGuardMember(id: string, guildId: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    memberId: `member-${id}`,
    memberName: `Member ${id}`,
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
    ...overrides,
  }
}

function createReview(id: string, guildId: string) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    memberId: `member-${id}`,
    actionType: 'kick',
    status: 'pending',
    reason: 'reason',
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt: new Date('2026-04-23T07:40:00.000Z'),
    updatedAt: new Date('2026-04-23T07:40:00.000Z'),
  }
}

function createReport(id: string, guildId: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    reporterMemberId: `reporter-${id}`,
    targetMemberId: `target-${id}`,
    reason: 'reason',
    aiStatus: 'completed',
    aiSeverity: 'medium',
    aiSummary: 'summary',
    createdAt: new Date('2026-04-23T07:45:00.000Z'),
    updatedAt: new Date('2026-04-23T07:45:00.000Z'),
    ...overrides,
  }
}

function createEvent(id: string, guildId: string, level: 'info' | 'medium' | 'high' | 'critical', payload: Record<string, unknown> = {}) {
  return {
    id,
    platform: 'onebot',
    botSelfId: 'bot',
    guildId,
    channelId: guildId,
    memberId: 'member',
    type: 'report_created',
    level,
    summary: 'summary',
    payload,
    createdAt: new Date('2026-04-23T07:50:00.000Z'),
    updatedAt: new Date('2026-04-23T07:50:00.000Z'),
  }
}

function createGuardTemplate(id: string) {
  return {
    id,
    name: `Template ${id}`,
    muteDurationSeconds: 600,
    kickAfterMinutes: 30,
    reminderTemplate: '请先完成认证',
    exemptUsers: [],
    enabled: true,
    createdAt: new Date('2026-04-23T07:00:00.000Z'),
    updatedAt: new Date('2026-04-23T07:00:00.000Z'),
  }
}

function createGuardBinding(id: string, guildId: string) {
  return {
    id,
    platform: 'onebot',
    guildId,
    templateId: 'tpl-1',
    enabled: true,
    note: null,
    createdAt: new Date('2026-04-23T07:05:00.000Z'),
    updatedAt: new Date('2026-04-23T07:05:00.000Z'),
  }
}

function createCommandPolicy(commandId: string) {
  return {
    commandId,
    roles: ['moderator'],
    minAuthority: 4,
    createdAt: new Date('2026-04-23T07:10:00.000Z'),
    updatedAt: new Date('2026-04-23T07:10:00.000Z'),
  }
}

function createProfile(qqID: string, verificationState: 'verified' | 'bound_unverified' | 'unbound') {
  return {
    qqID,
    userID: Number(qqID),
    boundAt: '2026-04-23T07:00:00.000Z',
    verificationState,
    profileVerificationStatus: verificationState === 'verified' ? 'verified' : 'pending',
    studentVerified: verificationState === 'verified',
  }
}
