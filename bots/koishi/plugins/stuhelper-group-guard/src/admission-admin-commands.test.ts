import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context, Session } from 'koishi'

import type { ModerationStore } from '@stuhelper/koishi-moderation-core'
import type {
  AdmissionRuntimeSettingsStore,
  AdmissionSession,
  GuardMemberRecord,
  GuardMemberStore,
  GuardPolicyStore,
  PlatformClient,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import { PlatformAPIError, resolveGroupGuardMessages } from '@stuhelper/koishi-shared'

import { registerAdmissionAdminCommands } from './admission-admin-commands'

const MESSAGES = resolveGroupGuardMessages()

type CommandHandler = (
  argv: { session?: Session },
  qqID?: string,
) => Promise<string | void>

interface HarnessOverrides {
  platform?: Partial<PlatformClient>
  guardStore?: Partial<GuardMemberStore>
  moderationStore?: Partial<ModerationStore>
  runtimeSettings?: Partial<AdmissionRuntimeSettingsStore>
  coordinator?: ReturnType<typeof createCoordinatorSpy>
}

function createHarness(overrides: HarnessOverrides = {}) {
  const commands = new Map<string, CommandHandler>()
  const ctx = {
    command: (definition: string) => {
      const name = definition.split(' ')[0]
      return {
        action: (handler: CommandHandler) => {
          commands.set(name, handler)
        },
      }
    },
  } as unknown as Context

  const coordinator = overrides.coordinator ?? createCoordinatorSpy()
  registerAdmissionAdminCommands(ctx, {
    platform: fakePlatform(overrides.platform),
    guardStore: fakeGuardStore(overrides.guardStore),
    policyStore: fakePolicyStore(),
    moderationStore: fakeModerationStore(overrides.moderationStore),
    config: createConfig(),
    runtimeSettings: fakeRuntimeSettings(overrides.runtimeSettings),
    admissionSubjectCoordinator: coordinator as never,
  })

  return {
    coordinator,
    run: (name: string, session: Session | undefined, qqID?: string) => {
      const handler = commands.get(name)
      assert.ok(handler, `command ${name} not registered`)
      return handler({ session }, qqID)
    },
  }
}

test('查询入群认证返回会话摘要', async () => {
  const harness = createHarness()
  const reply = await harness.run('查询入群认证', createSession(), '2001')

  assert.ok(typeof reply === 'string')
  assert.match(reply, /2001/)
})

test('入群认证命令被 WebUI 关闭时返回停用提示', async () => {
  const harness = createHarness({
    runtimeSettings: { isAdmissionCommandsEnabled: async () => false },
  })
  const reply = await harness.run('查询入群认证', createSession(), '2001')

  assert.equal(reply, MESSAGES.admissionCommandsDisabled)
})

test('权限不足时拒绝执行', async () => {
  const harness = createHarness()
  const session = createSession({ user: { authority: 1 } })
  const reply = await harness.run('清空入群未认证次数', session, '2001')

  assert.equal(reply, MESSAGES.commandAccessDenied)
})

test('重发认证链接在去重窗口内只调一次后端', async () => {
  let calls = 0
  const harness = createHarness({
    platform: {
      resendAdmissionSessionLink: async () => {
        calls += 1
        return createAdmissionSession()
      },
    },
  })
  const session = createSession()

  await harness.run('重发认证链接', session, '2001')
  const second = await harness.run('重发认证链接', session, '2001')

  assert.equal(calls, 1)
  assert.equal(second, undefined)
})

test('重发认证链接后端失败时回滚去重，可立即重试', async () => {
  let calls = 0
  const harness = createHarness({
    platform: {
      resendAdmissionSessionLink: async () => {
        calls += 1
        if (calls === 1) {
          throw new PlatformAPIError('backend down', 502, 'internal')
        }
        return createAdmissionSession()
      },
    },
  })
  const session = createSession()

  const first = await harness.run('重发认证链接', session, '2001')
  await harness.run('重发认证链接', session, '2001')

  assert.ok(typeof first === 'string', '失败时应返回错误提示而不是抛出')
  assert.equal(calls, 2, 'forget 后重试必须再次命中后端')
})

test('重新生成认证链接成功时重置禁言、同步本地并记录提醒事件', async () => {
  const mutes: Array<{ guildId: string; memberId: string; duration: number }> = []
  const synced: string[] = []
  const events: Array<{ sessionID: string; action: string }> = []
  const harness = createHarness({
    platform: {
      regenerateAdmissionSessionLink: async () => ({
        session: createAdmissionSession({
          initialMuteUntil: new Date(Date.now() + 60_000).toISOString(),
        }),
        token: 'abc',
        authURL: 'https://join.stuhelper.com/verify/abc',
      }),
      recordAdmissionEvent: async (sessionID, input) => {
        events.push({ sessionID, action: input.action })
      },
    },
    guardStore: {
      findActiveBySubject: async () => createMember(),
      markBackendSynced: async (id) => {
        synced.push(id)
      },
      markReminderSent: async () => {},
    },
  })
  const session = createSession({ mutes })

  const reply = await harness.run('重新生成认证链接', session, '2001')

  assert.equal(reply, undefined, '成功路径由提醒投递收尾，不返回文本')
  assert.equal(mutes.length, 1)
  assert.ok(mutes[0].duration > 0)
  assert.ok(synced.length >= 1)
  assert.deepEqual(events, [{ sessionID: 'session-1', action: 'remind' }])
})

test('重新生成认证链接后端失败时回滚去重，可立即重试', async () => {
  let calls = 0
  const harness = createHarness({
    platform: {
      regenerateAdmissionSessionLink: async () => {
        calls += 1
        throw new PlatformAPIError('backend down', 502, 'internal')
      },
    },
  })
  const session = createSession()

  const first = await harness.run('重新生成认证链接', session, '2001')
  await harness.run('重新生成认证链接', session, '2001')

  assert.ok(typeof first === 'string')
  assert.equal(calls, 2, 'forget 后重试必须再次命中后端')
})

test('重新生成认证链接发现已 verified 时解除禁言并释放本地记录', async () => {
  const mutes: Array<{ guildId: string; memberId: string; duration: number }> = []
  const released: string[] = []
  const harness = createHarness({
    platform: {
      regenerateAdmissionSessionLink: async () => ({
        session: createAdmissionSession({ status: 'verified' }),
        token: 'abc',
        authURL: 'https://join.stuhelper.com/verify/abc',
      }),
      recordAdmissionEvent: async () => {},
    },
    guardStore: {
      findActiveBySubject: async () => createMember(),
      markBackendSynced: async () => {},
      markReleased: async (id) => {
        released.push(id)
      },
    },
  })
  const session = createSession({ mutes })

  const reply = await harness.run('重新生成认证链接', session, '2001')

  assert.ok(typeof reply === 'string')
  assert.match(reply, /已完成 StuHelper 学生身份认证/)
  assert.deepEqual(mutes.map((item) => item.duration), [0])
  assert.deepEqual(released, ['qq:2118785781:178037297:2001'])
})

test('跳过入群认证按取消→独占→清理的协调顺序执行', async () => {
  const coordinator = createCoordinatorSpy()
  const harness = createHarness({
    coordinator,
    guardStore: {
      findActiveBySubject: async () => createMember(),
      markBackendSynced: async () => {},
      markReleased: async () => {},
    },
  })
  const session = createSession()

  const reply = await harness.run('跳过入群认证', session, '2001')

  assert.ok(typeof reply === 'string')
  assert.match(reply, /跳过/)
  assert.deepEqual(coordinator.calls, [
    'cancelSubject',
    'cancel',
    'runExclusive',
    'clearSubjectCancellation',
  ])
})

test('跳过入群认证解禁失败时容忍并返回失败提示', async () => {
  const harness = createHarness({
    guardStore: {
      findActiveBySubject: async () => createMember(),
      markBackendSynced: async () => {},
      markReleased: async () => {},
    },
  })
  const session = createSession({
    muteGuildMember: async () => {
      throw new Error('mute api down')
    },
  })

  const reply = await harness.run('跳过入群认证', session, '2001')

  assert.ok(typeof reply === 'string')
  assert.match(reply, /解除禁言失败|mute api down/)
})

test('跳过入群认证后端失败时清理取消标记并回滚去重', async () => {
  let calls = 0
  const coordinator = createCoordinatorSpy()
  const harness = createHarness({
    coordinator,
    platform: {
      skipAdmissionSessionForMember: async () => {
        calls += 1
        throw new PlatformAPIError('backend down', 502, 'internal')
      },
      getAdmissionSessionByMember: async () => createAdmissionSession(),
    },
  })
  const session = createSession()

  const first = await harness.run('跳过入群认证', session, '2001')
  await harness.run('跳过入群认证', session, '2001')

  assert.ok(typeof first === 'string')
  assert.equal(calls, 2, 'forget 后重试必须再次命中后端')
  assert.equal(
    coordinator.calls.filter((name) => name === 'clearSubjectCancellation').length,
    2,
    '每次失败都必须清理 subject 取消标记',
  )
})

test('跳过入群认证遇 409 且会话已取消时复用该会话', async () => {
  const harness = createHarness({
    platform: {
      skipAdmissionSessionForMember: async () => {
        throw new PlatformAPIError('invalid state', 409, 'admission.invalid_state')
      },
      getAdmissionSessionByMember: async () =>
        createAdmissionSession({ status: 'cancelled' }),
    },
    guardStore: {
      findActiveBySubject: async () => createMember(),
      markBackendSynced: async () => {},
      markReleased: async () => {},
    },
  })
  const session = createSession()

  const reply = await harness.run('跳过入群认证', session, '2001')

  assert.ok(typeof reply === 'string')
  assert.match(reply, /跳过/)
})

test('清空入群未认证次数返回原次数', async () => {
  const harness = createHarness()
  const reply = await harness.run('清空入群未认证次数', createSession(), '2001')

  assert.ok(typeof reply === 'string')
  assert.match(reply, /已清空 QQ 2001/)
  assert.match(reply, /原次数：2/)
})

test('解除入群拉黑成功与 404 各自返回对应提示', async () => {
  const okHarness = createHarness()
  const okReply = await okHarness.run('解除入群拉黑', createSession(), '2001')
  assert.ok(typeof okReply === 'string')
  assert.match(okReply, /已解除 QQ 2001/)

  const missingHarness = createHarness({
    platform: {
      releaseMemberBlacklistBySubject: async () => {
        throw new PlatformAPIError('not found', 404, 'not_found')
      },
    },
  })
  const missingReply = await missingHarness.run('解除入群拉黑', createSession(), '2001')
  assert.ok(typeof missingReply === 'string')
  assert.match(missingReply, /没有活动入群拉黑记录/)
})

function createCoordinatorSpy() {
  const calls: string[] = []
  return {
    calls,
    cancelSubject: () => {
      calls.push('cancelSubject')
    },
    cancel: () => {
      calls.push('cancel')
    },
    runExclusive: async <T>(_ref: unknown, run: () => Promise<T>) => {
      calls.push('runExclusive')
      return run()
    },
    clearSubjectCancellation: () => {
      calls.push('clearSubjectCancellation')
    },
  }
}

function createSession(overrides: {
  user?: { authority?: number }
  mutes?: Array<{ guildId: string; memberId: string; duration: number }>
  muteGuildMember?: (guildId: string, memberId: string, duration: number) => Promise<void>
} = {}): Session {
  const mutes = overrides.mutes ?? []
  return {
    platform: 'onebot',
    guildId: '178037297',
    channelId: '178037297',
    userId: '9001',
    selfId: '2118785781',
    user: overrides.user ?? { authority: 4 },
    bot: {
      platform: 'onebot',
      muteGuildMember:
        overrides.muteGuildMember ??
        (async (guildId: string, memberId: string, duration: number) => {
          mutes.push({ guildId, memberId, duration })
        }),
      sendPrivateMessage: async () => ({}),
    },
    send: async () => {},
  } as unknown as Session
}

function fakeRuntimeSettings(overrides: Partial<AdmissionRuntimeSettingsStore> = {}) {
  return {
    isAdmissionCommandsEnabled: async () => true,
    getAdmissionReminderDeliveryConfig: async () => ({
      groupEnabled: true,
      directEnabled: false,
    }),
    ...overrides,
  } as unknown as AdmissionRuntimeSettingsStore
}

function fakeGuardStore(overrides: Partial<GuardMemberStore> = {}) {
  return {
    findActiveBySubject: async () => null,
    markBackendSynced: async () => {},
    markReleased: async () => {},
    markReminderSent: async () => {},
    ...overrides,
  } as unknown as GuardMemberStore
}

function fakePolicyStore() {
  return {
    resolvePolicy: async () => ({
      id: 'default',
      enabled: true,
    }),
  } as unknown as GuardPolicyStore
}

function fakeModerationStore(overrides: Partial<ModerationStore> = {}) {
  return {
    getCommandPolicy: async () => null,
    getMemberRoles: async () => [],
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
    skipAdmissionSessionForMember: async () =>
      createAdmissionSession({ status: 'cancelled' }),
    resetAdmissionFailureCount: async () => ({
      platform: 'qq',
      guildID: '178037297',
      qqID: '2001',
      previousFailureCount: 2,
    }),
    releaseMemberBlacklistBySubject: async () => ({ id: 'blacklist-1' }),
    recordAdmissionEvent: async () => {},
    ...overrides,
  } as unknown as PlatformClient
}

function createConfig(): StuhelperGroupGuardPluginConfig {
  return {
    platform: {
      baseUrl: 'https://stuhelper.com',
      serviceToken: 'secret-token',
    },
    scheduler: {
      scanIntervalSeconds: 300,
    },
    actionStream: {
      reconnectDelaySeconds: 5,
    },
    retention: {
      messageRetentionDays: 14,
      eventRetentionDays: 30,
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
    admissionSessionID: 'session-1',
    backendSyncPending: false,
    joinedAt: new Date('2026-06-04T08:00:00.000Z'),
    deadlineAt: new Date('2026-06-04T09:00:00.000Z'),
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
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
