import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import type {
  AdmissionSession,
  AdmissionRuntimeSettingsStore,
  GuardPolicyStore,
  PlatformClient,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

import {
  buildAdmissionRuntimePageData,
  handleAdmissionRuntimeAction,
  registerAdmissionConsoleAPI,
} from './admission-console-api'
import type { GuardMemberRecord } from './model'
import type { GuardMemberStore } from './store'

test('admission runtime page data redacts service token and exposes guard state', async () => {
  const data = await buildAdmissionRuntimePageData(fakeContext(), {
    config: createConfig(),
    platform: fakePlatform(),
    runtimeSettings: fakeRuntimeSettings(),
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
  assert.deepEqual(data.activeMembers[0].availableActions, [
    'query',
    'reset-failures',
    'release-blacklist',
    'regenerate',
    'skip',
  ])
})

test('admission runtime resend action calls backend, sends reminder, and records bot event', async () => {
  const sentMessages: Array<{ channelId: string; message: string }> = []
  const reminderMarks: Array<{ id: string; now: Date }> = []
  const recordedEvents: Array<{ sessionID: string; messageID?: string }> = []
  const session = createAdmissionSession()
  const data = await handleAdmissionRuntimeAction(
    fakeContext(sentMessages),
    {
      config: createConfig(),
      platform: fakePlatform({
        async resendAdmissionSessionLink(input) {
          assert.deepEqual(input, {
            platform: 'qq',
            guildID: '178037297',
            qqID: '2001',
          })
          return session
        },
        async recordAdmissionEvent(sessionID, input) {
          assert.equal(input.action, 'remind')
          assert.equal(input.success, true)
          recordedEvents.push({ sessionID, messageID: input.messageID })
        },
      }),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID(id) {
          assert.equal(id, 'qq:2118785781:178037297:2001')
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
        async markReminderSent(id, now) {
          reminderMarks.push({ id, now })
        },
      }),
      policyStore: fakePolicyStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'resend' },
    { auth: { id: 42 } },
  )

  assert.equal(data, '已重发 QQ 2001 的入群认证链接。')
  assert.equal(sentMessages.length, 1)
  assert.equal(sentMessages[0].channelId, '178037297')
  assert.match(sentMessages[0].message, /https:\/\/join\.stuhelper\.com\/verify\/abc/)
  assert.equal(reminderMarks[0].id, 'qq:2118785781:178037297:2001')
  assert.deepEqual(recordedEvents, [{ sessionID: 'session-1', messageID: 'message-1' }])
})

test('admission runtime resend does not record backend event after losing the active record', async () => {
  const sentMessages: Array<{ channelId: string; message: string }> = []
  let eventRecorded = false
  const data = await handleAdmissionRuntimeAction(
    fakeContext(sentMessages),
    {
      config: createConfig(),
      platform: fakePlatform({
        async recordAdmissionEvent() {
          eventRecorded = true
        },
      }),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
        async markReminderSent() {
          return false
        },
      }),
      policyStore: fakePolicyStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'resend' },
    { auth: { id: 42 } },
  )

  assert.equal(data, '入群认证记录已被其他任务处理，请刷新页面后确认当前状态。')
  assert.equal(sentMessages.length, 1)
  assert.equal(eventRecorded, false)
})

test('admission runtime regenerate does not record verified release after losing the active record', async () => {
  const sentMessages: Array<{ channelId: string; message: string }> = []
  const muteActions: Array<{ guildId: string; memberId: string; duration: number }> = []
  let eventRecorded = false
  const data = await handleAdmissionRuntimeAction(
    fakeContext(sentMessages, muteActions),
    {
      config: createConfig(),
      platform: fakePlatform({
        async regenerateAdmissionSessionLink() {
          return {
            session: createAdmissionSession({ status: 'verified' }),
            token: 'abc',
            authURL: 'https://join.stuhelper.com/verify/abc',
          }
        },
        async recordAdmissionEvent() {
          eventRecorded = true
        },
      }),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
        async markBackendSynced() {
          return false
        },
      }),
      policyStore: fakePolicyStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'regenerate' },
    { auth: { id: 42 } },
  )

  assert.equal(data, '入群认证记录已被其他任务处理，请刷新页面后确认当前状态。')
  assert.equal(sentMessages.length, 0)
  assert.deepEqual(muteActions, [{ guildId: '178037297', memberId: '2001', duration: 0 }])
  assert.equal(eventRecorded, false)
})

test('admission runtime settings action persists WebUI switch changes', async () => {
  const listeners = new Map<string, (input: unknown) => Promise<string>>()
  const savedInputs: unknown[] = []
  let refreshCount = 0
  registerAdmissionConsoleAPI({
    ...fakeContext(),
    console: {
      addListener(event: string, listener: (input: unknown) => Promise<string>) {
        listeners.set(event, listener)
      },
    },
  } as unknown as Context, {
    config: createConfig(),
    platform: fakePlatform(),
    runtimeSettings: fakeRuntimeSettings({
      async saveSettings(input) {
        savedInputs.push(input)
        return {
          id: 'default',
          actionStreamEnabled: false,
          publicCommandsEnabled: false,
          admissionCommandsEnabled: true,
          moderationEnabled: true,
          freshmanForwardEnabled: false,
          fallbackScanEnabled: false,
          createdAt: new Date('2026-06-04T07:00:00.000Z'),
          updatedAt: new Date('2026-06-04T08:00:00.000Z'),
        }
      },
    }),
    guardStore: fakeGuardStore(),
    policyStore: fakePolicyStore(),
    onRuntimeSettingsChanged: async () => {
      refreshCount += 1
    },
  })

  const listener = listeners.get('stuhelperGroupGuard/action/save-admission-runtime-settings')
  assert.ok(listener)
  const result = await listener({ actionStreamEnabled: false, moderationEnabled: true, fallbackScanEnabled: false, ignored: 'x' })
  assert.equal(result, '已保存入群认证运行开关。')
  assert.equal(refreshCount, 1)
  assert.deepEqual(savedInputs, [{
    actionStreamEnabled: false,
    publicCommandsEnabled: undefined,
    admissionCommandsEnabled: undefined,
    moderationEnabled: true,
    freshmanForwardEnabled: undefined,
    fallbackScanEnabled: false,
  }])
})

function fakeContext(
  sentMessages: Array<{ channelId: string; message: string }> = [],
  muteActions: Array<{ guildId: string; memberId: string; duration: number }> = [],
) {
  return {
    bots: [{
      platform: 'onebot',
      selfId: '2118785781',
      status: 'online',
      async sendMessage(channelId: string, message: string) {
        sentMessages.push({ channelId, message })
        return 'message-1'
      },
      async muteGuildMember(guildId: string, memberId: string, duration: number) {
        muteActions.push({ guildId, memberId, duration })
      },
    }],
  } as unknown as Context
}

function fakeRuntimeSettings(overrides: Partial<AdmissionRuntimeSettingsStore> = {}) {
  return {
    getSettings: async () => ({
      id: 'default',
      actionStreamEnabled: true,
      publicCommandsEnabled: false,
      admissionCommandsEnabled: true,
      moderationEnabled: false,
      freshmanForwardEnabled: false,
      fallbackScanEnabled: true,
      createdAt: new Date('2026-06-04T07:00:00.000Z'),
      updatedAt: new Date('2026-06-04T07:00:00.000Z'),
    }),
    saveSettings: async () => ({
      id: 'default',
      actionStreamEnabled: true,
      publicCommandsEnabled: false,
      admissionCommandsEnabled: true,
      moderationEnabled: false,
      freshmanForwardEnabled: false,
      fallbackScanEnabled: true,
      createdAt: new Date('2026-06-04T07:00:00.000Z'),
      updatedAt: new Date('2026-06-04T07:00:00.000Z'),
    }),
    ...overrides,
  } as unknown as AdmissionRuntimeSettingsStore
}

function fakeGuardStore(overrides: Partial<GuardMemberStore> = {}) {
  return {
    listActive: async () => [createMember()],
    getActiveByID: async () => createMember({ backendSyncPending: false, admissionSessionID: 'session-1' }),
    markReminderSent: async () => {},
    markBackendSynced: async () => {},
    markReleased: async () => {},
    markLastError: async () => {},
    ...overrides,
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

function fakePlatform(overrides: Partial<PlatformClient> = {}) {
  return {
    getAdmissionSessionByMember: async () => createAdmissionSession(),
    resendAdmissionSessionLink: async () => createAdmissionSession(),
    regenerateAdmissionSessionLink: async () => ({
      session: createAdmissionSession(),
      token: 'abc',
      authURL: 'https://join.stuhelper.com/verify/abc',
    }),
    skipAdmissionSessionForMember: async () => createAdmissionSession({ status: 'verified' }),
    resetAdmissionFailureCount: async () => ({
      platform: 'qq',
      guildID: '178037297',
      qqID: '2001',
      previousFailureCount: 2,
    }),
    releaseMemberBlacklistBySubject: async () => ({
      id: 'blacklist-1',
    }),
    recordAdmissionEvent: async () => {},
    ...overrides,
  } as unknown as PlatformClient
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

function createMember(overrides: Partial<GuardMemberRecord> = {}): GuardMemberRecord {
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
    ...overrides,
  }
}

function createAdmissionSession(overrides: Partial<AdmissionSession> = {}): AdmissionSession {
  return {
    id: 'session-1',
    platform: 'qq',
    guildID: '178037297',
    channelID: '178037297',
    qqID: '2001',
    userID: null,
    status: 'joined_muted',
    tokenExpiresAt: '2026-06-04T09:00:00.000Z',
    linkWaitDeadlineAt: '2026-06-04T09:00:00.000Z',
    submissionWaitDeadlineAt: '2026-06-04T10:00:00.000Z',
    manualReviewDeadlineAt: null,
    initialMuteUntil: '2026-06-04T09:00:00.000Z',
    projectionPending: false,
    authURL: 'https://join.stuhelper.com/verify/abc',
    maxMaterialBytes: 10_000_000,
    lastBotError: null,
    failureCount: 1,
    remainingRetryCount: 1,
    willBlacklistOnTimeout: false,
    ...overrides,
  }
}
