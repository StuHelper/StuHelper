import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import type { ModerationStore } from '@stuhelper/koishi-moderation-core'
import type {
  AdmissionSession,
  AdmissionRuntimeSettingsStore,
  GuardMemberRecord,
  GuardMemberStore,
  GuardPolicyStore,
  PlatformClient,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import { PlatformAPIError, resolveGroupGuardMessages } from '@stuhelper/koishi-shared'

import {
  buildAdmissionRuntimePageData,
  handleAdmissionRuntimeAction,
  registerAdmissionConsoleAPI,
} from './admission-console-api'

test('admission runtime page data redacts service token and exposes guard state', async () => {
  const data = await buildAdmissionRuntimePageData(fakeContext(), {
    config: createConfig(),
    platform: fakePlatform(),
    runtimeSettings: fakeRuntimeSettings(),
    guardStore: fakeGuardStore(),
    policyStore: fakePolicyStore(),
    moderationStore: fakeModerationStore(),
  })

  assert.equal(data.platform.baseUrl, 'https://stuhelper.com')
  assert.equal(data.platform.serviceTokenConfigured, true)
  assert.doesNotMatch(JSON.stringify(data), /secret-token/)
  assert.equal(data.stats.enabledBindingCount, 1)
  assert.equal(data.stats.activeMemberCount, 1)
  assert.equal(data.stats.backendSyncPendingCount, 1)
  assert.equal(data.moderation.keywordRuleCount, 2)
  assert.equal(data.bots[0].platform, 'onebot')
  assert.equal(data.activeMembers[0].memberId, '2001')
  assert.equal(data.commands.adminCommandsEnabled, true)
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
      moderationStore: fakeModerationStore(),
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

test('admission runtime resend action respects direct-only reminder delivery', async () => {
  const sentMessages: Array<{ channelId: string; message: string }> = []
  const privateMessages: Array<{ userId: string; message: string; guildId?: string }> = []
  const recordedEvents: Array<{ sessionID: string; messageID?: string }> = []
  const data = await handleAdmissionRuntimeAction(
    fakeContext(sentMessages, [], privateMessages),
    {
      config: createConfig(),
      platform: fakePlatform({
        async recordAdmissionEvent(sessionID, input) {
          recordedEvents.push({ sessionID, messageID: input.messageID })
        },
      }),
      runtimeSettings: fakeRuntimeSettings({
        getAdmissionReminderDeliveryConfig: async () => ({
          groupEnabled: false,
          directEnabled: true,
        }),
      }),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
      }),
      policyStore: fakePolicyStore(),
      moderationStore: fakeModerationStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'resend' },
    { auth: { id: 42 } },
  )

  assert.equal(data, '已重发 QQ 2001 的入群认证链接。')
  assert.deepEqual(sentMessages, [])
  assert.equal(privateMessages.length, 1)
  assert.equal(privateMessages[0].userId, '2001')
  assert.equal(privateMessages[0].guildId, '178037297')
  assert.match(privateMessages[0].message, /https:\/\/join\.stuhelper\.com\/verify\/abc/)
  assert.deepEqual(recordedEvents, [{ sessionID: 'session-1', messageID: 'direct-message-1' }])
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
      moderationStore: fakeModerationStore(),
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
      moderationStore: fakeModerationStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'regenerate' },
    { auth: { id: 42 } },
  )

  assert.equal(data, '入群认证记录已被其他任务处理，请刷新页面后确认当前状态。')
  assert.equal(sentMessages.length, 0)
  assert.deepEqual(muteActions, [{ guildId: '178037297', memberId: '2001', duration: 0 }])
  assert.equal(eventRecorded, false)
})

test('admission runtime skip keeps local release when QQ unmute fails', async () => {
  const synced: unknown[] = []
  const released: Array<{ id: string; now: Date }> = []
  const data = await handleAdmissionRuntimeAction(
    fakeContext([], [], [], {
      async muteGuildMember() {
        throw new Error('Error with request set_group_ban, retcode: 1200')
      },
    }),
    {
      config: createConfig(),
      platform: fakePlatform({
        async skipAdmissionSessionForMember(input) {
          assert.equal(input.operatorQQID, 'console:42')
          return createAdmissionSession({ id: 'session-skipped', status: 'cancelled' })
        },
      }),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
        async markBackendSynced(id, input) {
          synced.push({ id, input })
        },
        async markReleased(id, now) {
          released.push({ id, now })
        },
      }),
      policyStore: fakePolicyStore(),
      moderationStore: fakeModerationStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'skip' },
    { auth: { id: 42 } },
  )

  assert.match(data, /已跳过 QQ 2001 在本群的入群认证/)
  assert.match(data, /自动解除禁言失败/)
  assert.match(data, /机器人缺少群管理员权限/)
  assert.equal(synced.length, 1)
  assert.equal(released.length, 1)
})

test('admission runtime skip cleans local record when backend session was already cancelled', async () => {
  const muteActions: Array<{ guildId: string; memberId: string; duration: number }> = []
  let queried = false
  let released = false
  const data = await handleAdmissionRuntimeAction(
    fakeContext([], muteActions),
    {
      config: createConfig(),
      platform: fakePlatform({
        async skipAdmissionSessionForMember() {
          throw new PlatformAPIError('invalid admission state', 409)
        },
        async getAdmissionSessionByMember(input) {
          queried = true
          assert.deepEqual(input, {
            platform: 'qq',
            guildID: '178037297',
            qqID: '2001',
          })
          return createAdmissionSession({ id: 'session-skipped', status: 'cancelled' })
        },
      }),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
        async markReleased() {
          released = true
        },
      }),
      policyStore: fakePolicyStore(),
      moderationStore: fakeModerationStore(),
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'skip' },
    { auth: { id: 42 } },
  )

  assert.equal(data, '已跳过 QQ 2001 在本群的入群认证并解除禁言。')
  assert.equal(queried, true)
  assert.equal(released, true)
  assert.deepEqual(muteActions, [{ guildId: '178037297', memberId: '2001', duration: 0 }])
})

test('admission runtime release blacklist reports missing blacklist records separately', async () => {
  await assert.rejects(
    () => handleAdmissionRuntimeAction(
      fakeContext(),
      {
        config: createConfig(),
        platform: fakePlatform({
          async releaseMemberBlacklistBySubject() {
            throw new PlatformAPIError('blacklist not found', 404)
          },
        }),
        runtimeSettings: fakeRuntimeSettings(),
        guardStore: fakeGuardStore({
          async getActiveByID() {
            return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
          },
        }),
        policyStore: fakePolicyStore(),
        moderationStore: fakeModerationStore(),
      },
      { recordId: 'qq:2118785781:178037297:2001', action: 'release-blacklist' },
      { auth: { id: 42 } },
    ),
    /QQ 2001 在本群没有活动入群拉黑记录/,
  )
})

test('admission runtime platform 404 names the missing admission session for session actions', async () => {
  await assert.rejects(
    () => handleAdmissionRuntimeAction(
      fakeContext(),
      {
        config: createConfig(),
        platform: fakePlatform({
          async getAdmissionSessionByMember() {
            throw new PlatformAPIError('admission session not found', 404)
          },
        }),
        runtimeSettings: fakeRuntimeSettings(),
        guardStore: fakeGuardStore({
          async getActiveByID() {
            return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
          },
        }),
        policyStore: fakePolicyStore(),
        moderationStore: fakeModerationStore(),
      },
      { recordId: 'qq:2118785781:178037297:2001', action: 'query' },
      { auth: { id: 42 } },
    ),
    /平台侧未找到 QQ 2001 在本群的入群认证会话/,
  )
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
          adminCommandsEnabled: true,
          admissionCommandsEnabled: true,
          moderationEnabled: true,
          freshmanForwardEnabled: false,
          fallbackScanEnabled: false,
          reminderGroupEnabled: true,
          reminderDirectEnabled: false,
          timeCodeReminderEnabled: false,
          createdAt: new Date('2026-06-04T07:00:00.000Z'),
          updatedAt: new Date('2026-06-04T08:00:00.000Z'),
        }
      },
    }),
    guardStore: fakeGuardStore(),
    policyStore: fakePolicyStore(),
    moderationStore: fakeModerationStore(),
    onRuntimeSettingsChanged: async () => {
      refreshCount += 1
    },
  })

  const listener = listeners.get('stuhelperGroupGuard/action/save-admission-runtime-settings')
  assert.ok(listener)
  const result = await listener({
    actionStreamEnabled: false,
    moderationEnabled: true,
    fallbackScanEnabled: false,
    timeCodeReminderEnabled: false,
    ignored: 'x',
  })
  assert.equal(result, '已保存入群认证运行开关。')
  assert.equal(refreshCount, 1)
  assert.deepEqual(savedInputs, [{
    actionStreamEnabled: false,
    publicCommandsEnabled: undefined,
    adminCommandsEnabled: undefined,
    admissionCommandsEnabled: undefined,
    moderationEnabled: true,
    freshmanForwardEnabled: undefined,
    fallbackScanEnabled: false,
    reminderGroupEnabled: undefined,
    reminderDirectEnabled: undefined,
    timeCodeReminderEnabled: false,
  }])
})

test('admission runtime console actions use configured message templates', async () => {
  const messageProvider = () => resolveGroupGuardMessages({
      admissionConsoleResendSuccess: 'Console 已重发 {qqID}',
      admissionQuerySummary: 'Console 查询 {qqID}/{statusLabel}/{nextStep}',
      admissionStatusLinked: '自定义已绑定',
      admissionNextStepLinked: '自定义下一步',
    })

  const queryResult = await handleAdmissionRuntimeAction(
    fakeContext(),
    {
      config: createConfig(),
      platform: fakePlatform({
        async getAdmissionSessionByMember() {
          return createAdmissionSession({ status: 'linked', userID: 42 })
        },
      }),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
      }),
      policyStore: fakePolicyStore(),
      moderationStore: fakeModerationStore(),
      messageProvider,
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'query' },
    { auth: { id: 42 } },
  )

  const resendResult = await handleAdmissionRuntimeAction(
    fakeContext(),
    {
      config: createConfig(),
      platform: fakePlatform(),
      runtimeSettings: fakeRuntimeSettings(),
      guardStore: fakeGuardStore({
        async getActiveByID() {
          return createMember({ backendSyncPending: false, admissionSessionID: 'session-1' })
        },
      }),
      policyStore: fakePolicyStore(),
      moderationStore: fakeModerationStore(),
      messageProvider,
    },
    { recordId: 'qq:2118785781:178037297:2001', action: 'resend' },
    { auth: { id: 42 } },
  )

  assert.equal(queryResult, 'Console 查询 2001/自定义已绑定/自定义下一步')
  assert.equal(resendResult, 'Console 已重发 2001')
})

function fakeContext(
  sentMessages: Array<{ channelId: string; message: string }> = [],
  muteActions: Array<{ guildId: string; memberId: string; duration: number }> = [],
  privateMessages: Array<{ userId: string; message: string; guildId?: string }> = [],
  botOverrides: Partial<{
    sendMessage(channelId: string, message: string): Promise<string | string[]>
    muteGuildMember(guildId: string, memberId: string, duration: number): Promise<void>
    getFriendList(): Promise<{ data: unknown[] }>
    sendPrivateMessage(userId: string, message: string, guildId?: string): Promise<string[]>
  }> = {},
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
      async getFriendList() {
        return { data: [] }
      },
      async sendPrivateMessage(userId: string, message: string, guildId?: string) {
        privateMessages.push({ userId, message, guildId })
        return ['direct-message-1']
      },
      ...botOverrides,
    }],
  } as unknown as Context
}

function fakeRuntimeSettings(overrides: Partial<AdmissionRuntimeSettingsStore> = {}) {
  return {
    getSettings: async () => ({
      id: 'default',
      actionStreamEnabled: true,
      publicCommandsEnabled: false,
      adminCommandsEnabled: true,
      admissionCommandsEnabled: true,
      moderationEnabled: false,
      freshmanForwardEnabled: false,
      fallbackScanEnabled: true,
      reminderGroupEnabled: true,
      reminderDirectEnabled: false,
      timeCodeReminderEnabled: true,
      createdAt: new Date('2026-06-04T07:00:00.000Z'),
      updatedAt: new Date('2026-06-04T07:00:00.000Z'),
    }),
    saveSettings: async () => ({
      id: 'default',
      actionStreamEnabled: true,
      publicCommandsEnabled: false,
      adminCommandsEnabled: true,
      admissionCommandsEnabled: true,
      moderationEnabled: false,
      freshmanForwardEnabled: false,
      fallbackScanEnabled: true,
      reminderGroupEnabled: true,
      reminderDirectEnabled: false,
      timeCodeReminderEnabled: true,
      createdAt: new Date('2026-06-04T07:00:00.000Z'),
      updatedAt: new Date('2026-06-04T07:00:00.000Z'),
    }),
    getAdmissionReminderDeliveryConfig: async () => ({
      groupEnabled: true,
      directEnabled: false,
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

function fakeModerationStore(overrides: Partial<ModerationStore> = {}) {
  return {
    listAllKeywordRules: async () => [
      {
        id: 'keyword-global',
        guildId: '*',
        pattern: 'spam',
        matchMode: 'includes',
        action: 'warn',
        enabled: true,
        muteSeconds: 0,
        note: null,
        createdAt: new Date('2026-06-04T07:00:00.000Z'),
        updatedAt: new Date('2026-06-04T07:00:00.000Z'),
      },
      {
        id: 'keyword-local',
        guildId: '178037297',
        pattern: 'ad',
        matchMode: 'includes',
        action: 'delete',
        enabled: true,
        muteSeconds: 0,
        note: null,
        createdAt: new Date('2026-06-04T07:00:00.000Z'),
        updatedAt: new Date('2026-06-04T07:00:00.000Z'),
      },
    ],
    ...overrides,
  } as unknown as ModerationStore
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
    scheduler: {
      scanIntervalSeconds: 300,
    },
    actionStream: {
      reconnectDelaySeconds: 5,
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
