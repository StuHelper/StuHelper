import assert from 'node:assert/strict'
import test from 'node:test'

import { buildDashboardPageData, DashboardPageService } from './dashboard-page.service'

test('buildDashboardPageData aggregates pending work', () => {
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
  })

  assert.equal(data.overview.pendingReviews, 1)
  assert.equal(data.overview.pendingAdmissions, 1)
  assert.equal(data.overview.openReports, 1)
  assert.equal(data.recentEvents[0].summary, '高危事件')
})

test('getOverviewData returns the same counts as the full dashboard but no detail lists', async () => {
  const deps = {
    loadPendingMembers: async () => [createGuardMember({ id: 'gm-1' })],
    loadPendingReviews: async () => [createReview()],
    loadRecentEvents: async () => [createEvent()],
    loadRecentReports: async () => [createReport()],
    loadCommandPolicies: async () => [createCommandPolicy()],
    loadGuardTemplates: async () => [createTemplate()],
    loadGuardBindings: async () => [createBinding()],
  }
  const service = new DashboardPageService(deps)

  const [full, overview] = await Promise.all([service.getPageData(), service.getOverviewData()])

  // 计数必须与全量 dashboard 的 overview 完全一致（脉冲徽标不得与控制台显示漂移）。
  assert.deepEqual(overview.overview, full.overview)
  assert.equal(typeof overview.generatedAt, 'string')
  // 轻量端点只返回计数，绝不携带任何明细列表。
  assert.deepEqual(Object.keys(overview).sort(), ['generatedAt', 'overview'])
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
