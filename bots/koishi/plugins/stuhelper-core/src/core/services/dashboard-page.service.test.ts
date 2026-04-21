import assert from 'node:assert/strict'
import test from 'node:test'

import { buildDashboardPageData } from './dashboard-page.service'

test('buildDashboardPageData aggregates pending work and system status', () => {
  const data = buildDashboardPageData({
    generatedAt: '2026-04-21T08:00:00.000Z',
    pendingMembers: [
      createGuardMember({
        id: 'gm-1',
        guildId: '1001',
        memberId: '2001',
      }),
    ],
    pendingReviews: [createReview()],
    recentEvents: [createEvent()],
    recentReports: [createReport()],
    commandPolicies: [createCommandPolicy()],
    guardTemplates: [createTemplate()],
    guardBindings: [createBinding()],
    moduleStates: [
      { name: 'status', description: '系统状态', state: 'loaded' },
      { name: 'report', description: '举报模块', state: 'error', error: 'boom' },
    ],
  })

  assert.equal(data.overview.pendingReviews, 1)
  assert.equal(data.overview.pendingAdmissions, 1)
  assert.equal(data.overview.openReports, 1)
  assert.equal(data.systemStatus[1].state, 'error')
  assert.equal(data.recentEvents[0].summary, '高危事件')
})

function createGuardMember(overrides: Partial<Parameters<typeof buildDashboardPageData>[0]['pendingMembers'][number]> = {}) {
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
    ...overrides,
  }
}

function createReview() {
  return {
    id: 'rv-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2002',
    actionType: 'kick' as const,
    status: 'pending' as const,
    reason: '命中复核规则',
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt: new Date('2026-04-21T07:55:00.000Z'),
    updatedAt: new Date('2026-04-21T07:55:00.000Z'),
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
    type: 'review_created' as const,
    level: 'high' as const,
    summary: '高危事件',
    payload: { reviewId: 'rv-1' },
    createdAt: new Date('2026-04-21T07:56:00.000Z'),
    updatedAt: new Date('2026-04-21T07:56:00.000Z'),
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
    aiSeverity: 'medium' as const,
    aiSummary: null,
    createdAt: new Date('2026-04-21T07:57:00.000Z'),
    updatedAt: new Date('2026-04-21T07:57:00.000Z'),
  }
}

function createCommandPolicy() {
  return {
    commandId: 'report',
    roles: ['admin'],
    minAuthority: 3,
    createdAt: new Date('2026-04-21T07:40:00.000Z'),
    updatedAt: new Date('2026-04-21T07:40:00.000Z'),
  }
}

function createTemplate() {
  return {
    id: 'tpl-1',
    name: '默认模板',
    muteDurationSeconds: 1800,
    kickAfterMinutes: 30,
    reminderTemplate: '请先完成认证',
    exemptUsers: [],
    enabled: true,
    createdAt: new Date('2026-04-21T07:30:00.000Z'),
    updatedAt: new Date('2026-04-21T07:30:00.000Z'),
  }
}

function createBinding() {
  return {
    id: 'onebot:1001',
    platform: 'onebot',
    guildId: '1001',
    templateId: 'tpl-1',
    enabled: true,
    note: null,
    createdAt: new Date('2026-04-21T07:35:00.000Z'),
    updatedAt: new Date('2026-04-21T07:35:00.000Z'),
  }
}
