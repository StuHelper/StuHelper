import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildAdmissionRuntimeModel,
  type AdmissionRuntimePageData,
} from './admission-runtime'

test('buildAdmissionRuntimeModel exposes admission runtime metrics and switch states', () => {
  const model = buildAdmissionRuntimeModel(createAdmissionRuntimeFixture())

  assert.equal(model.metrics[0].label, '目标群')
  assert.equal(model.metrics[0].value, 2)
  assert.equal(model.metrics[0].note, '2 个启用绑定，去重后 2 个有效目标群')
  assert.equal(model.metrics[1].value, 2)
  assert.equal(model.metrics[3].tone, 'danger')
  assert.equal(model.switchRows.find((row) => row.id === 'service-token')?.tone, 'success')
  assert.equal(model.switchRows.find((row) => row.id === 'action-stream')?.editable, true)
  assert.equal(model.switchRows.find((row) => row.id === 'action-stream')?.settingKey, 'actionStreamEnabled')
  assert.equal(model.switchRows.find((row) => row.id === 'fallback-scan')?.tone, 'warning')
  assert.equal(model.switchRows.find((row) => row.id === 'admin-commands')?.settingKey, 'adminCommandsEnabled')
  assert.equal(
    model.switchRows.find((row) => row.id === 'admission-commands')?.note,
    '权限由命令策略 admission-admin 控制',
  )
  assert.equal(model.switchRows.find((row) => row.id === 'reminder-group')?.settingKey, 'reminderGroupEnabled')
  assert.equal(
    model.switchRows.find((row) => row.id === 'reminder-group')?.note,
    '只影响学生认证链接群内投递，不影响验证码提醒',
  )
  assert.equal(model.switchRows.find((row) => row.id === 'reminder-direct')?.settingKey, 'reminderDirectEnabled')
  assert.equal(
    model.switchRows.find((row) => row.id === 'reminder-direct')?.note,
    '只影响学生认证链接私聊 / 临时会话投递',
  )
  assert.equal(model.switchRows.find((row) => row.id === 'time-code-reminder')?.settingKey, 'timeCodeReminderEnabled')
  assert.equal(
    model.switchRows.find((row) => row.id === 'time-code-reminder')?.note,
    '入群后发送验证码规则提示；关闭后仍会校验验证码并超时踢出',
  )
  assert.deepEqual(model.activeMembers.map((member) => member.memberId), ['2001', '2002'])
  assert.deepEqual(model.activeMembers[0].availableActions, ['query', 'resend'])
})

test('buildAdmissionRuntimeModel hides bot-wide controls from guild-scoped page data', () => {
  const data = createAdmissionRuntimeFixture()
  data.globalRuntime = null

  const model = buildAdmissionRuntimeModel(data)

  assert.deepEqual(model.switchRows, [])
  assert.equal(model.metrics[0].value, 2)
  assert.deepEqual(model.activeMembers.map((member) => member.memberId), ['2001', '2002'])
})

function createAdmissionRuntimeFixture(): AdmissionRuntimePageData {
  return {
    generatedAt: '2026-06-04T08:00:00.000Z',
    globalRuntime: {
      platform: {
        baseUrl: 'https://stuhelper.com',
        serviceTokenConfigured: true,
      },
      scheduler: {
        fallbackScanEnabled: true,
        scanIntervalSeconds: 300,
      },
      actionStream: {
        enabled: true,
        reconnectDelaySeconds: 5,
      },
      commands: {
        publicCommandsRegistered: true,
        publicCommandsEnabled: false,
        adminCommandsRegistered: true,
        adminCommandsEnabled: true,
        admissionCommandsRegistered: true,
        admissionCommandsEnabled: true,
      },
      moderation: {
        enabled: false,
        keywordRuleCount: 0,
        repeatThreshold: 3,
        repeatWindowSize: 3,
        antiRecallNotify: false,
      },
      freshmanForward: {
        enabled: false,
      },
      reminderDelivery: {
        groupEnabled: true,
        directEnabled: false,
      },
      timeCode: {
        reminderEnabled: true,
      },
      bots: [{ platform: 'onebot', selfId: '2118785781', status: 'online' }],
    },
    stats: {
      templateCount: 1,
      bindingCount: 2,
      enabledBindingCount: 2,
      activeMemberCount: 2,
      backendSyncPendingCount: 1,
      membersWithAdmissionSessionCount: 1,
      membersWithLastErrorCount: 1,
    },
    templates: [{
      id: 'default',
      name: '默认模板',
      enabled: true,
      muteDurationSeconds: 2592000,
      kickAfterMinutes: 60,
      exemptUserCount: 0,
      updatedAt: '2026-06-04T07:00:00.000Z',
    }],
    bindings: [
      {
        id: 'qq:178037297',
        platform: 'qq',
        guildId: '178037297',
        templateId: 'default',
        kickAfterMinutes: 8,
        kickAfterMinutesOverride: 8,
        enabled: true,
        note: null,
        updatedAt: '2026-06-04T07:00:00.000Z',
      },
      {
        id: 'qq:1001',
        platform: 'qq',
        guildId: '1001',
        templateId: 'default',
        kickAfterMinutes: 60,
        kickAfterMinutesOverride: null,
        enabled: true,
        note: null,
        updatedAt: '2026-06-04T07:00:00.000Z',
      },
    ],
    activeMembers: [
      createMember('2002', '2026-06-04T08:20:00.000Z', true, null, 'failed'),
      createMember('2001', '2026-06-04T08:10:00.000Z', false, 'session-1', null),
    ],
  }
}

function createMember(
  memberId: string,
  deadlineAt: string,
  backendSyncPending: boolean,
  admissionSessionID: string | null,
    lastError: string | null,
) {
  return {
    id: `qq:bot:178037297:${memberId}`,
    platform: 'qq',
    botSelfId: '2118785781',
    guildId: '178037297',
    channelId: '178037297',
    memberId,
    memberName: memberId,
    verificationState: 'pending',
    admissionSessionID,
    backendSyncPending,
    joinedAt: '2026-06-04T08:00:00.000Z',
    deadlineAt,
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: null,
    reminderSentAt: null,
    lastError,
    availableActions: ['query', 'resend'],
  }
}
