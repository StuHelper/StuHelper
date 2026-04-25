import assert from 'node:assert/strict'
import test from 'node:test'

import { buildReviewPageData } from './review-page.service'

test('buildReviewPageData maps reviews, admissions and reports into work items', () => {
  const data = buildReviewPageData({
    generatedAt: '2026-04-21T08:00:00.000Z',
    pendingReviews: [createReview()],
    pendingMembers: [createGuardMember()],
    reports: [createReport()],
    events: [createEvent()],
  })

  assert.equal(data.items.length, 3)
  assert.equal(data.items[0].kind, 'report')
  assert.ok(data.items.some((item) => item.kind === 'review' && item.availableActions.includes('execute')))
  assert.ok(data.items.some((item) => item.kind === 'admission' && item.availableActions.includes('approve')))
  assert.deepEqual(data.items.find((item) => item.kind === 'report')?.relatedEventIds, ['evt-1'])
})

test('buildReviewPageData links report events that still use legacy reportID payload field', () => {
  const data = buildReviewPageData({
    generatedAt: '2026-04-21T08:00:00.000Z',
    pendingReviews: [],
    pendingMembers: [],
    reports: [createReport()],
    events: [createLegacyReportEvent()],
  })

  assert.deepEqual(data.items.find((item) => item.kind === 'report')?.relatedEventIds, ['evt-legacy'])
})

function createReview() {
  return {
    id: 'rv-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2002',
    actionType: 'kick_and_block' as const,
    status: 'pending' as const,
    reason: '高风险发言',
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt: new Date('2026-04-21T07:55:00.000Z'),
    updatedAt: new Date('2026-04-21T07:55:00.000Z'),
  }
}

function createGuardMember() {
  return {
    id: 'gm-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2001',
    memberName: 'Alice',
    verificationState: 'bound_unverified',
    joinedAt: new Date('2026-04-21T07:50:00.000Z'),
    deadlineAt: new Date('2026-04-21T08:20:00.000Z'),
    mutedAt: new Date('2026-04-21T07:51:00.000Z'),
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: new Date('2026-04-21T07:50:00.000Z'),
    updatedAt: new Date('2026-04-21T07:52:00.000Z'),
  }
}

function createReport() {
  return {
    id: 'rp-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    reporterMemberId: '2003',
    targetMemberId: '2002',
    reason: '恶意刷屏',
    aiStatus: 'pending' as const,
    aiSeverity: 'high' as const,
    aiSummary: null,
    createdAt: new Date('2026-04-21T07:57:00.000Z'),
    updatedAt: new Date('2026-04-21T07:57:00.000Z'),
  }
}

function createEvent() {
  return {
    id: 'evt-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2002',
    type: 'report_created' as const,
    level: 'high' as const,
    summary: '举报已创建',
    payload: { reportId: 'rp-1' },
    createdAt: new Date('2026-04-21T07:58:00.000Z'),
    updatedAt: new Date('2026-04-21T07:58:00.000Z'),
  }
}

function createLegacyReportEvent() {
  return {
    id: 'evt-legacy',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '9999',
    channelId: '9999',
    memberId: '9999',
    type: 'report_created' as const,
    level: 'high' as const,
    summary: '遗留举报事件',
    payload: { reportID: 'rp-1' },
    createdAt: new Date('2026-04-21T07:58:00.000Z'),
    updatedAt: new Date('2026-04-21T07:58:00.000Z'),
  }
}
