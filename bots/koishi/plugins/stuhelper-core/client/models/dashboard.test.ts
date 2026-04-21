import assert from 'node:assert/strict'
import test from 'node:test'

import type { DashboardPageData } from '../page-types'
import { buildDashboardModel } from './dashboard'

test('buildDashboardModel aggregates pending rows and fixed shortcuts', () => {
  const model = buildDashboardModel(createDashboardFixture())

  assert.equal(model.todoRows.length, 2)
  assert.equal(model.todoRows[0].target.view, 'review')
  assert.equal(model.todoRows[1].target.view, 'identity')
  assert.equal(model.shortcuts[2].target.view, 'config')
  assert.equal(model.shortcuts[2].target.workspace, 'bindings')
  assert.equal(model.activityRows[0].id, 'evt-1')
})

function createDashboardFixture(): DashboardPageData {
  return {
    generatedAt: '2026-04-21T08:00:00.000Z',
    overview: {
      pendingReviews: 1,
      pendingAdmissions: 1,
      openReports: 1,
      highRiskEvents: 1,
      policyItems: 3,
    },
    pendingMembers: [
      {
        id: 'gm-1',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        memberId: '2001',
        memberName: 'Alice',
        verificationState: 'bound_unverified',
        joinedAt: '2026-04-21T07:50:00.000Z',
        deadlineAt: '2026-04-21T08:20:00.000Z',
        mutedAt: '2026-04-21T07:51:00.000Z',
        reminderSentAt: null,
        releasedAt: null,
        kickedAt: null,
        lastError: null,
        createdAt: '2026-04-21T07:50:00.000Z',
        updatedAt: '2026-04-21T07:52:00.000Z',
      },
    ],
    pendingReviews: [
      {
        id: 'rv-1',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        memberId: '2002',
        actionType: 'kick',
        status: 'pending',
        reason: '刷屏',
        operatorMemberId: null,
        resolutionNote: null,
        payload: null,
        createdAt: '2026-04-21T07:55:00.000Z',
        updatedAt: '2026-04-21T07:55:00.000Z',
      },
    ],
    recentEvents: [
      {
        id: 'evt-1',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        memberId: '2002',
        type: 'review_created',
        level: 'high',
        summary: '已创建复核',
        payload: null,
        createdAt: '2026-04-21T07:58:00.000Z',
        updatedAt: '2026-04-21T07:58:00.000Z',
      },
    ],
    recentReports: [
      {
        id: 'rp-1',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        reporterMemberId: '2003',
        targetMemberId: '2002',
        reason: '广告',
        aiStatus: 'pending',
        aiSeverity: 'high',
        aiSummary: '疑似广告',
        createdAt: '2026-04-21T07:57:00.000Z',
        updatedAt: '2026-04-21T07:57:00.000Z',
      },
    ],
    commandPolicies: [],
    guardTemplates: [],
    guardBindings: [],
    systemStatus: [],
  }
}
