import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import type {
  GuardPolicyStore,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

import { buildAdmissionRuntimePageData } from './admission-console-api'
import type { GuardMemberRecord } from './model'
import type { GuardMemberStore } from './store'

test('admission runtime page data redacts service token and exposes guard state', async () => {
  const data = await buildAdmissionRuntimePageData(fakeContext(), {
    config: createConfig(),
    guardStore: fakeGuardStore(),
    policyStore: fakePolicyStore(),
  })

  assert.equal(data.platform.baseUrl, 'https://stuhelper.com')
  assert.equal(data.platform.serviceTokenConfigured, true)
  assert.doesNotMatch(JSON.stringify(data), /secret-token/)
  assert.equal(data.stats.targetGroupCount, 1)
  assert.equal(data.stats.activeMemberCount, 1)
  assert.equal(data.stats.backendSyncPendingCount, 1)
  assert.equal(data.bots[0].platform, 'onebot')
  assert.equal(data.activeMembers[0].memberId, '2001')
})

function fakeContext() {
  return {
    bots: [{ platform: 'onebot', selfId: '2118785781', status: 'online' }],
  } as unknown as Context
}

function fakeGuardStore() {
  return {
    listActive: async () => [createMember()],
  } as unknown as GuardMemberStore
}

function fakePolicyStore() {
  return {
    listTemplates: async () => [{
      id: 'default',
      name: '默认模板',
      enabled: true,
      muteDurationSeconds: 2592000,
      kickAfterMinutes: 60,
      reminderTemplate: '请先认证',
      exemptUsers: [],
      createdAt: new Date('2026-06-04T07:00:00.000Z'),
      updatedAt: new Date('2026-06-04T07:00:00.000Z'),
    }],
    listBindings: async () => [{
      id: 'qq:178037297',
      platform: 'qq',
      guildId: '178037297',
      templateId: 'default',
      enabled: true,
      note: null,
      createdAt: new Date('2026-06-04T07:00:00.000Z'),
      updatedAt: new Date('2026-06-04T07:00:00.000Z'),
    }],
  } as unknown as GuardPolicyStore
}

function createConfig(): StuhelperGroupGuardPluginConfig {
  return {
    platform: {
      baseUrl: 'https://user:pass@stuhelper.com/',
      serviceToken: 'secret-token',
    },
    guard: {
      targetGroups: ['178037297'],
      muteDurationSeconds: 2592000,
      kickAfterMinutes: 60,
      reminderTemplate: '请先认证',
      exemptUsers: [],
    },
    scheduler: {
      fallbackScanEnabled: true,
      scanIntervalSeconds: 300,
    },
    actionStream: {
      enabled: true,
      reconnectDelaySeconds: 5,
    },
    moderation: {
      enabled: false,
      repeatThreshold: 3,
      repeatWindowSize: 3,
      warningThresholdExpression: 'warnings >= 3',
      defaultMuteSeconds: 600,
      antiRecallNotify: false,
      keywordRules: [],
    },
    fun: {
      diceSides: 100,
      muteLotteryBaseSeconds: 120,
      muteLotteryMaxSeconds: 600,
      muteLotteryPityThreshold: 5,
      muteLotteryPitySeconds: 300,
    },
    ai: {
      enabled: false,
      endpoint: '',
      apiKey: '',
      model: '',
    },
    commands: {
      enabled: false,
    },
    admissionCommands: {
      enabled: true,
      minAuthority: 4,
      operatorQQIDs: [],
    },
    freshmanForward: {
      enabled: false,
    },
  }
}

function createMember(): GuardMemberRecord {
  return {
    id: 'qq:2118785781:178037297:2001',
    platform: 'qq',
    botSelfId: '2118785781',
    guildId: '178037297',
    channelId: '178037297',
    memberId: '2001',
    memberName: '2001',
    verificationState: 'pending',
    admissionSessionID: null,
    backendSyncPending: true,
    joinedAt: new Date('2026-06-04T08:00:00.000Z'),
    deadlineAt: new Date('2026-06-04T09:00:00.000Z'),
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: 'backend unavailable',
    createdAt: new Date('2026-06-04T08:00:00.000Z'),
    updatedAt: new Date('2026-06-04T08:00:00.000Z'),
  }
}
