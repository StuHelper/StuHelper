import assert from 'node:assert/strict'
import test from 'node:test'

import type { StuhelperConsoleData } from '../../src/console-types'
import { buildDashboardModel } from './model'

test('buildDashboardModel returns metrics and route targets in the fixed order', () => {
  const model = buildDashboardModel(
    createConsoleData({
      pendingReviews: [
        createReview('rv_1', '2026-04-20T09:30:00.000Z'),
        createReview('rv_2', '2026-04-20T09:20:00.000Z'),
      ],
      pendingMembers: [
        createMember('gm_1', '2026-04-20T11:00:00.000Z'),
        createMember('gm_2', '2026-04-20T12:00:00.000Z'),
      ],
      keywordRules: Array.from({ length: 2 }, (_, index) => createKeywordRule(`kw_${index + 1}`)),
      commandPolicies: [createCommandPolicy('report')],
      memberRoles: [createMemberRole('mr_1')],
      guardTemplates: [createGuardTemplate('guard_1')],
      guardBindings: [createGuardBinding('binding_1')],
      recentReports: [
        createReport('rp_1', '2026-04-20T09:40:00.000Z'),
        createReport('rp_2', '2026-04-20T09:10:00.000Z'),
      ],
    }),
  )

  assert.deepEqual(
    model.metrics.map((item) => [item.label, item.value]),
    [
      ['待复核', 2],
      ['身份认证', 2],
      ['最近举报', 2],
      ['策略项', 6],
    ],
  )
  assert.equal(model.todoRows[0].kind, 'review')
  assert.equal(model.todoRows[0].target.section, 'enforcement')
  assert.equal(model.todoRows[0].target.queue, 'review')
  assert.equal(model.todoRows[0].target.id, 'rv_1')
  assert.equal(model.todoRows[0].target.source, 'dashboard')
  assert.equal(model.todoRows[1].target.section, 'enforcement')
  assert.equal(model.todoRows[1].target.id, 'rv_2')

  const identityRow = model.todoRows.find((item) => item.kind === 'identity')
  assert.ok(identityRow)
  assert.equal(identityRow.target.section, 'identity')
  assert.equal(identityRow.target.queue, 'member')
  assert.equal(identityRow.target.id, 'gm_1')
  assert.equal(model.shortcuts[1].label, '处理待认证成员')
  assert.equal(model.shortcuts[1].target.section, 'identity')
  assert.equal(model.shortcuts[2].target.section, 'policy')
  assert.equal(model.shortcuts[2].target.queue, 'guard-bindings')
  assert.deepEqual(
    model.policySummary.map((item) => [item.label, item.value]),
    [
      ['关键词规则', 2],
      ['命令权限', 1],
      ['成员角色', 1],
      ['守卫模板', 1],
      ['群绑定', 1],
    ],
  )
})

test('buildDashboardModel sorts recentActivity by timestamp desc and limits it to six rows', () => {
  const model = buildDashboardModel(
    createConsoleData({
      recentEvents: [
        createEvent('evt_1', '2026-04-20T09:15:00.000Z'),
        createEvent('evt_2', '2026-04-20T09:45:00.000Z'),
        createEvent('evt_3', '2026-04-20T08:55:00.000Z'),
      ],
      recentReports: [
        createReport('rp_1', '2026-04-20T09:50:00.000Z'),
        createReport('rp_2', '2026-04-20T09:35:00.000Z'),
        createReport('rp_3', '2026-04-20T09:05:00.000Z'),
        createReport('rp_4', '2026-04-20T08:45:00.000Z'),
      ],
    }),
  )

  assert.equal(model.recentActivity.length, 6)
  assert.deepEqual(
    model.recentActivity.map((item) => item.id),
    ['rp_1', 'evt_2', 'rp_2', 'evt_1', 'rp_3', 'evt_3'],
  )
  assert.deepEqual(
    model.recentActivity.map((item) => item.kind),
    ['report', 'event', 'report', 'event', 'report', 'event'],
  )
})

test('buildDashboardModel exposes dashboard status, system status, recent events and recent changes', () => {
  const model = buildDashboardModel(
    createConsoleData({
      generatedAt: '2026-04-20T10:30:00.000Z',
      pendingReviews: [createReview('rv_1', '2026-04-20T09:30:00.000Z')],
      pendingMembers: [createMember('gm_1', '2026-04-20T11:00:00.000Z')],
      recentEvents: [
        createEvent('evt_1', '2026-04-20T10:05:00.000Z'),
        createEvent('evt_2', '2026-04-20T09:50:00.000Z'),
      ],
      keywordRules: [createKeywordRule('kw_1', '2026-04-20T10:20:00.000Z')],
      commandPolicies: [createCommandPolicy('report', '2026-04-20T10:10:00.000Z')],
      memberRoles: [createMemberRole('mr_1', '2026-04-20T10:00:00.000Z')],
      guardTemplates: [createGuardTemplate('guard_1', '2026-04-20T10:15:00.000Z')],
      guardBindings: [createGuardBinding('binding_1', '2026-04-20T10:25:00.000Z')],
    }),
  )

  assert.equal(model.statusBand.length, 3)
  assert.equal(model.statusBand[0]?.label, '最后同步')
  assert.equal(model.statusBand[1]?.label, '积压总量')
  assert.equal(model.systemStatus.length, 3)
  assert.equal(model.systemStatus[0]?.label, '数据服务')
  assert.deepEqual(
    model.recentEvents.map((item) => item.id),
    ['evt_1', 'evt_2'],
  )
  assert.deepEqual(
    model.recentChanges.map((item) => item.id),
    ['binding_1', 'kw_1', 'guard_1', 'report', 'mr_1'],
  )
})

function createConsoleData(
  overrides: Partial<StuhelperConsoleData> = {},
): StuhelperConsoleData {
  return {
    title: 'StuHelper 群管中心',
    generatedAt: '2026-04-20T10:00:00.000Z',
    supportedCommandIds: [],
    overview: {
      pendingReviews: overrides.pendingReviews?.length ?? 0,
      openReports: overrides.recentReports?.length ?? 0,
      warningMembers: overrides.pendingMembers?.length ?? 0,
      highRiskEvents: overrides.recentEvents?.length ?? 0,
    },
    pendingMembers: [],
    pendingReviews: [],
    keywordRules: [],
    commandPolicies: [],
    memberRoles: [],
    guardTemplates: [],
    guardBindings: [],
    recentEvents: [],
    recentReports: [],
    ...overrides,
  }
}

function createMember(id: string, deadlineAt: string) {
  return {
    id,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    memberId: `${id}-member`,
    memberName: `成员 ${id}`,
    verificationState: 'bound_unverified' as const,
    joinedAt: '2026-04-20T08:00:00.000Z',
    deadlineAt,
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: '2026-04-20T08:00:00.000Z',
    updatedAt: '2026-04-20T08:10:00.000Z',
  }
}

function createReview(id: string, createdAt: string) {
  return {
    id,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    memberId: `${id}-member`,
    actionType: 'kick' as const,
    status: 'pending' as const,
    reason: `复核事项 ${id}`,
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt,
    updatedAt: createdAt,
  }
}

function createEvent(id: string, createdAt: string) {
  return {
    id,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    memberId: `${id}-member`,
    type: 'keyword-hit',
    level: 'medium' as const,
    summary: `事件摘要 ${id}`,
    payload: null,
    createdAt,
    updatedAt: createdAt,
  }
}

function createReport(id: string, createdAt: string) {
  return {
    id,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    reporterMemberId: `${id}-reporter`,
    targetMemberId: `${id}-target`,
    reason: `举报原因 ${id}`,
    aiStatus: 'completed' as const,
    aiSeverity: 'high' as const,
    aiSummary: `举报摘要 ${id}`,
    createdAt,
    updatedAt: createdAt,
  }
}

function createKeywordRule(id: string, updatedAt = '2026-04-20T08:00:00.000Z') {
  return {
    id,
    guildId: 'guild-1',
    pattern: `rule-${id}`,
    matchMode: 'includes' as const,
    action: 'warn' as const,
    enabled: true,
    muteSeconds: 0,
    note: '',
    createdAt: '2026-04-20T08:00:00.000Z',
    updatedAt,
  }
}

function createCommandPolicy(commandId: string, updatedAt = '2026-04-20T08:00:00.000Z') {
  return {
    commandId,
    roles: ['admin'],
    minAuthority: 2,
    createdAt: '2026-04-20T08:00:00.000Z',
    updatedAt,
  }
}

function createMemberRole(id: string, updatedAt = '2026-04-20T08:00:00.000Z') {
  return {
    id,
    guildId: 'guild-1',
    memberId: `${id}-member`,
    roles: ['moderator'],
    createdAt: '2026-04-20T08:00:00.000Z',
    updatedAt,
  }
}

function createGuardTemplate(id: string, updatedAt = '2026-04-20T08:00:00.000Z') {
  return {
    id,
    name: `模板 ${id}`,
    muteDurationSeconds: 600,
    kickAfterMinutes: 30,
    reminderTemplate: '请完成认证',
    exemptUsers: [],
    enabled: true,
    createdAt: '2026-04-20T08:00:00.000Z',
    updatedAt,
  }
}

function createGuardBinding(id: string, updatedAt = '2026-04-20T08:00:00.000Z') {
  return {
    id,
    platform: 'mock',
    guildId: 'guild-1',
    templateId: 'guard_1',
    enabled: true,
    note: '',
    createdAt: '2026-04-20T08:00:00.000Z',
    updatedAt,
  }
}
