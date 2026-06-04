import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildAdmissionRuntimeModel,
  type AdmissionRuntimePageData,
} from './admission-runtime'

test('buildAdmissionRuntimeModel exposes admission runtime metrics and switch states', () => {
  const model = buildAdmissionRuntimeModel(createAdmissionRuntimeFixture())

  assert.equal(model.metrics[0].label, '目标群')
  assert.equal(model.metrics[0].value, 1)
  assert.equal(model.metrics[1].value, 2)
  assert.equal(model.metrics[3].tone, 'danger')
  assert.equal(model.switchRows.find((row) => row.id === 'service-token')?.tone, 'success')
  assert.equal(model.switchRows.find((row) => row.id === 'fallback-scan')?.tone, 'warning')
  assert.deepEqual(model.activeMembers.map((member) => member.memberId), ['2001', '2002'])
})

function createAdmissionRuntimeFixture(): AdmissionRuntimePageData {
  return {
    generatedAt: '2026-06-04T08:00:00.000Z',
    platform: {
      baseUrl: 'https://stuhelper.com',
      serviceTokenConfigured: true,
    },
    guard: {
      targetGroups: ['178037297'],
      muteDurationSeconds: 2592000,
      kickAfterMinutes: 60,
      reminderTemplate: '请先认证',
      exemptUserCount: 0,
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
      publicCommandsEnabled: false,
      admissionCommandsEnabled: true,
      admissionCommandMinAuthority: 4,
      admissionCommandOperatorQQIDCount: 0,
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
    bots: [{ platform: 'onebot', selfId: '2118785781', status: 'online' }],
    stats: {
      targetGroupCount: 1,
      templateCount: 1,
      bindingCount: 1,
      enabledBindingCount: 1,
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
    bindings: [{
      id: 'qq:178037297',
      platform: 'qq',
      guildId: '178037297',
      templateId: 'default',
      enabled: true,
      note: null,
      updatedAt: '2026-06-04T07:00:00.000Z',
    }],
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
  }
}
