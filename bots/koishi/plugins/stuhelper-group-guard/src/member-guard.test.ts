import assert from 'node:assert/strict'
import test from 'node:test'

import { PlatformAPIError } from '@stuhelper/koishi-shared'

import { AdmissionReminderDeduper } from './admission-reminder-deduper'
import { AdmissionSubjectCoordinator } from './admission-subject-coordinator'
import { MemberGuardService } from './member-guard'

test('member guard creates admission session, mutes, and sends canonical auth link', async () => {
  const savedRecords: unknown[] = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  const createSessionCalls: unknown[] = []
  const admissionEvents: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession(input: unknown) {
        createSessionCalls.push(input)
        return {
          token: 'token-1',
          authURL: 'https://join.stuhelper.com/verify/token-1',
          session: {
            id: 'session-1',
            platform: 'qq',
            guildID: 'guild-1',
            channelID: 'channel-1',
            qqID: '10001',
            status: 'joined_muted',
            tokenExpiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
            linkWaitDeadlineAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
            submissionWaitDeadlineAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
            initialMuteUntil: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
            projectionPending: false,
          },
        }
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: unknown) { savedRecords.push(record) },
      async markMuted() {},
      async markReminderSent() {},
    },
    policyStore: {
      async resolvePolicy() {
        return {
          source: 'static',
          templateId: 'static',
          joinHandlingStrategy: 'post_join_guard',
          exemptUsers: [],
        }
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'qq',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
        muteActions.push({ guildId, memberId, duration })
      },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-join']
      },
    },
  } as any)

  assert.deepEqual(createSessionCalls, [{
    platform: 'qq',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    botSelfID: '514',
  }])
  assert.equal(muteActions.length, 1)
  assert.equal(muteActions[0].guildId, 'guild-1')
  assert.equal(muteActions[0].memberId, '10001')
  assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
  assert.equal(savedRecords.length, 1)
  assert.match(JSON.stringify(savedRecords[0]), /session-1/)
  assert.match(sentMessages[0], /https:\/\/join\.stuhelper\.com\/verify\/token-1/)
  assert.doesNotMatch(sentMessages[0], /buaa\.team|sso\.stuhelper\.com/)
  assert.deepEqual(admissionEvents, [{
    sessionID: 'session-1',
    input: {
      action: 'remind',
      success: true,
      messageID: 'message-join',
    },
  }])
})

test('member guard suppresses post-join reminder when admission was skipped before mute', async () => {
  const created = admissionResult('session-skipped-before-mute', 'token-skipped-before-mute')
  const savedRecords: any[] = []
  const muteActions: unknown[] = []
  const sentMessages: string[] = []
  const admissionEvents: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        return created
      },
      async getAdmissionSessionByMember() {
        return {
          ...created.session,
          status: 'cancelled',
        }
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: activeRecordStore(savedRecords),
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded(memberAddedSession({
    bot: {
      muteGuildMember: async (...args: unknown[]) => { muteActions.push(args) },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-skipped']
      },
    },
  }))

  assert.equal(savedRecords.length, 1)
  assert.ok(savedRecords[0].releasedAt instanceof Date)
  assert.deepEqual(muteActions, [])
  assert.deepEqual(sentMessages, [])
  assert.deepEqual(admissionEvents, [])
})

test('member guard releases mute and suppresses post-join reminder when admission was skipped after mute', async () => {
  const created = admissionResult('session-skipped-after-mute', 'token-skipped-after-mute')
  const savedRecords: any[] = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  let statusChecks = 0
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        return created
      },
      async getAdmissionSessionByMember() {
        statusChecks += 1
        if (statusChecks === 1) return created.session
        return {
          ...created.session,
          status: 'cancelled',
        }
      },
      async recordAdmissionEvent() {
        throw new Error('skipped admission should not record reminder event')
      },
    },
    guardStore: activeRecordStore(savedRecords),
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded(memberAddedSession({
    bot: {
      muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
        muteActions.push({ guildId, memberId, duration })
      },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-skipped']
      },
    },
  }))

  assert.equal(statusChecks, 2)
  assert.equal(savedRecords.length, 1)
  assert.ok(savedRecords[0].mutedAt instanceof Date)
  assert.ok(savedRecords[0].releasedAt instanceof Date)
  assert.equal(muteActions.length, 2)
  assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
  assert.deepEqual(muteActions[1], { guildId: 'guild-1', memberId: '10001', duration: 0 })
  assert.deepEqual(sentMessages, [])
})

test('member guard suppresses post-join reminder when local skip cancellation wins the race', async () => {
  const created = admissionResult('session-locally-skipped', 'token-locally-skipped')
  const savedRecords: any[] = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  const admissionEvents: unknown[] = []
  const coordinator = new AdmissionSubjectCoordinator()
  let statusChecks = 0
  const subject = {
    platform: 'qq',
    botSelfId: '514',
    guildId: 'guild-1',
    memberId: '10001',
  }
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        return created
      },
      async getAdmissionSessionByMember() {
        statusChecks += 1
        if (statusChecks === 2) {
          coordinator.cancelSubject(subject)
          coordinator.cancel(subject, created.session.id)
        }
        return created.session
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: activeRecordStore(savedRecords),
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    admissionSubjectCoordinator: coordinator,
  } as any)

  await service.handleGuildMemberAdded(memberAddedSession({
    bot: {
      muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
        muteActions.push({ guildId, memberId, duration })
      },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-should-not-send']
      },
    },
  }))

  assert.equal(statusChecks, 2)
  assert.equal(savedRecords.length, 1)
  assert.ok(savedRecords[0].releasedAt instanceof Date)
  assert.equal(muteActions.length, 2)
  assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
  assert.deepEqual(muteActions[1], { guildId: 'guild-1', memberId: '10001', duration: 0 })
  assert.deepEqual(sentMessages, [])
  assert.deepEqual(admissionEvents, [])
})

test('member guard can deliver admission reminder by temporary direct session only', async () => {
  const groupMessages: string[] = []
  const privateMessages: Array<{ userId: string, content: string, guildId?: string }> = []
  const admissionEvents: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        return admissionResult('session-direct', 'token-direct')
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending() {},
      async markMuted() {},
      async markReminderSent() {},
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    admissionReminderDelivery: {
      groupEnabled: false,
      directEnabled: true,
    },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'onebot',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      muteGuildMember: async () => {},
      sendMessage: async (_channelId: string, content: string) => {
        groupMessages.push(content)
        throw new Error('group reminder should not be sent')
      },
      getFriendList: async () => ({ data: [] }),
      sendPrivateMessage: async (userId: string, content: string, guildId?: string) => {
        privateMessages.push({ userId, content, guildId })
        return ['direct-message-join']
      },
    },
  } as any)

  assert.deepEqual(groupMessages, [])
  assert.equal(privateMessages.length, 1)
  assert.equal(privateMessages[0].userId, '10001')
  assert.equal(privateMessages[0].guildId, 'guild-1')
  assert.match(privateMessages[0].content, /https:\/\/join\.stuhelper\.com\/verify\/token-direct/)
  assert.deepEqual(admissionEvents, [{
    sessionID: 'session-direct',
    input: {
      action: 'remind',
      success: true,
      messageID: 'direct-message-join',
    },
  }])
})

test('member guard skips mute and reminder when backend marks member verified', async () => {
  const savedRecords: unknown[] = []
  const muteActions: unknown[] = []
  const sentMessages: string[] = []
  const moderationEvents: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        const now = Date.now()
        return {
          token: 'token-verified',
          authURL: 'https://join.stuhelper.com/verify/token-verified',
          session: {
            id: 'session-verified',
            platform: 'qq',
            guildID: 'guild-1',
            channelID: 'channel-1',
            qqID: '10001',
            status: 'verified',
            tokenExpiresAt: new Date(now + 60 * 60 * 1000).toISOString(),
            tokenConsumedAt: new Date(now).toISOString(),
            linkWaitDeadlineAt: new Date(now + 60 * 60 * 1000).toISOString(),
            submissionWaitDeadlineAt: new Date(now + 24 * 60 * 60 * 1000).toISOString(),
            initialMuteUntil: new Date(now + 30 * 24 * 60 * 60 * 1000).toISOString(),
            verifiedAt: new Date(now).toISOString(),
            cancelledAt: new Date(now).toISOString(),
            projectionPending: false,
          },
        }
      },
    },
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: unknown) { savedRecords.push(record) },
    },
    policyStore: {
      async resolvePolicy() {
        return {
          source: 'static',
          templateId: 'static',
          joinHandlingStrategy: 'post_join_guard',
          exemptUsers: [],
        }
      },
    },
    moderationStore: { async appendEvent(event: unknown) { moderationEvents.push(event) } },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'onebot',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      muteGuildMember: async (...args: unknown[]) => { muteActions.push(args) },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-join']
      },
    },
  } as any)

  assert.deepEqual(savedRecords, [])
  assert.deepEqual(muteActions, [])
  assert.deepEqual(sentMessages, [])
  assert.match(JSON.stringify(moderationEvents[0]), /跳过入群禁言/)
})

test('member guard auto-approves join request through admission policy on onebot runtime', async () => {
  const decisions: unknown[] = []
  const events: unknown[] = []
  const approvals: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async resolveJoinRequestDecision(input: unknown) {
        decisions.push(input)
        return {
          decision: 'approve',
          reason: 'verified_auto_approve',
          verificationState: 'verified',
          joinHandlingStrategy: 'join_request_review',
          autoApproveVerifiedJoin: true,
          autoApproveUnverifiedJoin: true,
          policyID: 'policy-1',
          userID: '6',
        }
      },
      async recordJoinRequestEvent(input: unknown) {
        events.push(input)
      },
    },
    policyStore: policyStoreFor(['guild-1']),
    guardStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberRequest(joinRequestSession({
    bot: {
      handleGuildMemberRequest: async (requestID: string, approve: boolean, reason?: string) => {
        approvals.push({ requestID, approve, reason })
      },
    },
  }))

  assert.deepEqual(decisions, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    rawEvent: { comment: '申请入群' },
  }])
  assert.deepEqual(approvals, [{ requestID: 'request-1', approve: true, reason: undefined }])
  assert.deepEqual(events, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    decision: 'approve',
    success: true,
    rawEvent: { comment: '申请入群' },
  }])
})

test('member guard handles join request review without a local post-join guard binding', async () => {
  const approvals: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async resolveJoinRequestDecision() {
        return {
          decision: 'approve',
          reason: 'verified_auto_approve',
          verificationState: 'verified',
          joinHandlingStrategy: 'join_request_review',
          autoApproveVerifiedJoin: true,
          autoApproveUnverifiedJoin: false,
          policyID: 'policy-review',
        }
      },
      async recordJoinRequestEvent() {},
    },
    policyStore: {
      async resolvePolicy() {
        throw new Error('join request review should not depend on local post-join guard policy')
      },
    },
    guardStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberRequest(joinRequestSession({
    bot: {
      handleGuildMemberRequest: async (requestID: string, approve: boolean, reason?: string) => {
        approvals.push({ requestID, approve, reason })
      },
    },
  }))

  assert.deepEqual(approvals, [{ requestID: 'request-1', approve: true, reason: undefined }])
})

test('member guard rejects join request when admission policy decision rejects', async () => {
  const events: unknown[] = []
  const approvals: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async resolveJoinRequestDecision() {
        return {
          decision: 'reject',
          reason: '请先完成 StuHelper 学生认证后再申请入群。',
          verificationState: 'unverified',
          joinHandlingStrategy: 'join_request_review',
          autoApproveVerifiedJoin: true,
          autoApproveUnverifiedJoin: false,
          policyID: 'policy-1',
        }
      },
      async recordJoinRequestEvent(input: unknown) {
        events.push(input)
      },
    },
    policyStore: policyStoreFor(['guild-1']),
    guardStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberRequest(joinRequestSession({
    bot: {
      handleGuildMemberRequest: async (requestID: string, approve: boolean, reason?: string) => {
        approvals.push({ requestID, approve, reason })
      },
    },
  }))

  assert.deepEqual(approvals, [{
    requestID: 'request-1',
    approve: false,
    reason: '请先完成 StuHelper 学生认证后再申请入群。',
  }])
  assert.deepEqual(events, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    decision: 'reject',
    success: true,
    rawEvent: { comment: '申请入群' },
  }])
})

test('member guard records local post-join time-code challenge after member joins', async () => {
  const savedRecords: any[] = []
  const sentMessages: string[] = []
  const moderationEvents: any[] = []
  const reminderMarks: Array<{ id: string, now: Date }> = []
  const muteActions: unknown[] = []
  const now = new Date('2026-06-17T07:30:00.000Z')
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        throw new Error('post-join time-code strategy must not create admission sessions')
      },
    },
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: any) { savedRecords.push(record) },
      async markReminderSent(id: string, markedAt: Date) {
        reminderMarks.push({ id, now: markedAt })
        const record = savedRecords.find((item) => item.id === id)
        if (record) record.reminderSentAt = markedAt
      },
    },
    moderationStore: { async appendEvent(event: any) { moderationEvents.push(event) } },
    logger: { error() {}, warn() {} },
    now: () => now,
  } as any)

  await service.handleGuildMemberAdded(memberAddedSession({
    bot: {
      muteGuildMember: async (...args: unknown[]) => {
        muteActions.push(args)
      },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-code']
      },
    },
  }))

  assert.equal(savedRecords.length, 1)
  assert.equal(savedRecords[0].admissionSessionID, null)
  assert.equal(savedRecords[0].backendSyncPending, false)
  assert.equal(savedRecords[0].verificationState, 'unbound')
  assert.equal(savedRecords[0].deadlineAt.toISOString(), '2026-06-17T08:00:00.000Z')
  assert.deepEqual(muteActions, [])
  assert.equal(reminderMarks.length, 1)
  assert.equal(reminderMarks[0].id, savedRecords[0].id)
  assert.match(sentMessages[0], /请在 30 分钟内发送验证码/)
  assert.match(sentMessages[0], /群公告/)
  assert.doesNotMatch(sentMessages[0], /1531/)
  assert.equal(moderationEvents.length, 1)
  assert.equal(moderationEvents[0].type, 'join_guarded')
  assert.equal(moderationEvents[0].payload.joinHandlingStrategy, 'post_join_time_code')
  assert.equal('code' in moderationEvents[0].payload, false)
})

test('member guard can disable post-join time-code reminder without disabling challenge', async () => {
  const savedRecords: any[] = []
  const sentMessages: string[] = []
  const moderationEvents: any[] = []
  const reminderMarks: Array<{ id: string, now: Date }> = []
  const now = new Date('2026-06-17T07:30:00.000Z')
  const service = new MemberGuardService({
    platform: {},
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: any) { savedRecords.push(record) },
      async markReminderSent(id: string, markedAt: Date) {
        reminderMarks.push({ id, now: markedAt })
      },
    },
    moderationStore: { async appendEvent(event: any) { moderationEvents.push(event) } },
    logger: { error() {}, warn() {} },
    isTimeCodeReminderEnabled: async () => false,
    now: () => now,
  } as any)

  await service.handleGuildMemberAdded(memberAddedSession({
    bot: {
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-code']
      },
    },
  }))

  assert.equal(savedRecords.length, 1)
  assert.equal(savedRecords[0].deadlineAt.toISOString(), '2026-06-17T08:00:00.000Z')
  assert.deepEqual(sentMessages, [])
  assert.deepEqual(reminderMarks, [])
  assert.equal(moderationEvents.length, 1)
  assert.equal(moderationEvents[0].payload.joinHandlingStrategy, 'post_join_time_code')
  assert.equal(moderationEvents[0].payload.reminderSent, false)
})

test('member guard releases post-join time-code challenge on valid group message', async () => {
  const now = new Date('2026-06-17T07:30:00.000Z')
  const record = timeCodeRecord({ deadlineAt: new Date('2026-06-17T08:00:00.000Z') })
  const releases: Array<{ id: string, now: Date }> = []
  const sentMessages: string[] = []
  const moderationEvents: any[] = []
  const service = new MemberGuardService({
    platform: {},
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async findActiveBySubject() { return record },
      async markReleased(id: string, markedAt: Date) {
        releases.push({ id, now: markedAt })
        record.releasedAt = markedAt
        return true
      },
    },
    moderationStore: { async appendEvent(event: any) { moderationEvents.push(event) } },
    logger: { error() {}, warn() {} },
    now: () => now,
  } as any)

  const handled = await service.handleMessage(messageSession({
    content: '验证码：1531',
    bot: {
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-verified']
      },
    },
  }))

  assert.equal(handled, true)
  assert.deepEqual(releases, [{ id: record.id, now }])
  assert.match(sentMessages[0], /验证通过/)
  assert.equal(moderationEvents.length, 1)
  assert.equal(moderationEvents[0].type, 'join_released')
  assert.equal(moderationEvents[0].payload.joinHandlingStrategy, 'post_join_time_code')
})

test('member guard validates post-join time-code against user message timestamp with 30 second tolerance', async () => {
  const processingTime = new Date('2026-06-17T07:31:05.000Z')
  const userMessageTime = new Date('2026-06-17T07:30:35.000Z')
  const record = timeCodeRecord({ deadlineAt: new Date('2026-06-17T08:00:00.000Z') })
  const releases: Array<{ id: string, now: Date }> = []
  const service = new MemberGuardService({
    platform: {},
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async findActiveBySubject() { return record },
      async markReleased(id: string, markedAt: Date) {
        releases.push({ id, now: markedAt })
        record.releasedAt = markedAt
        return true
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    now: () => processingTime,
  } as any)

  const handled = await service.handleMessage(messageSession({
    content: '验证码：1531',
    timestamp: userMessageTime.getTime(),
    bot: { sendMessage: async () => ['message-verified'] },
  }))

  assert.equal(handled, true)
  assert.deepEqual(releases, [{ id: record.id, now: processingTime }])
})

test('member guard rejects post-join time-code outside 30 second tolerance', async () => {
  const processingTime = new Date('2026-06-17T07:31:35.000Z')
  const userMessageTime = new Date('2026-06-17T07:31:31.000Z')
  const record = timeCodeRecord({ deadlineAt: new Date('2026-06-17T08:00:00.000Z') })
  const sentMessages: string[] = []
  const service = new MemberGuardService({
    platform: {},
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async findActiveBySubject() { return record },
      async markReleased() {
        throw new Error('out-of-window code must not release guard record')
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    now: () => processingTime,
  } as any)

  const handled = await service.handleMessage(messageSession({
    content: '1531',
    timestamp: userMessageTime.getTime(),
    bot: {
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-invalid']
      },
    },
  }))

  assert.equal(handled, false)
  assert.deepEqual(sentMessages, ['<at id="10001"/> 验证码不正确，请核对后重新发送。'])
})

test('member guard keeps post-join time-code challenge active on invalid code', async () => {
  const now = new Date('2026-06-17T07:30:00.000Z')
  const record = timeCodeRecord({ deadlineAt: new Date('2026-06-17T08:00:00.000Z') })
  const sentMessages: string[] = []
  const service = new MemberGuardService({
    platform: {},
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async findActiveBySubject() { return record },
      async markReleased() {
        throw new Error('invalid code must not release guard record')
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    now: () => now,
  } as any)

  const handled = await service.handleMessage(messageSession({
    content: '9999',
    bot: {
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-invalid']
      },
    },
  }))

  assert.equal(handled, false)
  assert.deepEqual(sentMessages, ['<at id="10001"/> 验证码不正确，请核对后重新发送。'])
})

test('member guard kicks expired post-join time-code challenge during scheduled scan', async () => {
  const now = new Date('2026-06-17T08:01:00.000Z')
  const record = timeCodeRecord({ deadlineAt: new Date('2026-06-17T08:00:00.000Z') })
  const sentMessages: string[] = []
  const kicks: Array<{ guildId: string, memberId: string, permanent?: boolean }> = []
  const kickedMarks: Array<{ id: string, now: Date }> = []
  const moderationEvents: any[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() { return [] },
    },
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async listBackendSyncPending() { return [] },
      async listActive() { return [record] },
      async markKicked(id: string, markedAt: Date) {
        kickedMarks.push({ id, now: markedAt })
        record.kickedAt = markedAt
        return true
      },
      async markLastError(id: string, message: string, markedAt: Date) {
        record.lastError = `${id}:${message}:${markedAt.toISOString()}`
      },
    },
    moderationStore: { async appendEvent(event: any) { moderationEvents.push(event) } },
    logger: { error() {}, warn() {} },
    now: () => now,
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async (_channelId: string, content: string) => {
      sentMessages.push(content)
      return ['message-expired']
    },
    muteGuildMember: async () => {},
    kickGuildMember: async (guildId: string, memberId: string, permanent?: boolean) => {
      kicks.push({ guildId, memberId, permanent })
    },
  } as any])

  assert.deepEqual(sentMessages, ['<at id="10001"/> 入群验证码超时，机器人将自动移出群聊。'])
  assert.deepEqual(kicks, [{ guildId: 'guild-1', memberId: '10001', permanent: undefined }])
  assert.deepEqual(kickedMarks, [{ id: record.id, now }])
  assert.equal(moderationEvents.length, 1)
  assert.equal(moderationEvents[0].type, 'join_expired')
  assert.equal(moderationEvents[0].payload.joinHandlingStrategy, 'post_join_time_code')
})

test('member guard still kicks expired post-join time-code challenge when timeout notice fails', async () => {
  const now = new Date('2026-06-17T08:01:00.000Z')
  const record = timeCodeRecord({ deadlineAt: new Date('2026-06-17T08:00:00.000Z') })
  const kicks: Array<{ guildId: string, memberId: string, permanent?: boolean }> = []
  const errors: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() { return [] },
    },
    policyStore: timeCodePolicyStore(),
    guardStore: {
      async listBackendSyncPending() { return [] },
      async listActive() { return [record] },
      async markKicked(id: string, markedAt: Date) {
        record.kickedAt = markedAt
        return id === record.id
      },
      async markLastError(_id: string, message: string) {
        errors.push(message)
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    now: () => now,
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => {
      throw new Error('notice failed')
    },
    muteGuildMember: async () => {},
    kickGuildMember: async (guildId: string, memberId: string, permanent?: boolean) => {
      kicks.push({ guildId, memberId, permanent })
    },
  } as any])

  assert.deepEqual(errors, ['notice failed'])
  assert.deepEqual(kicks, [{ guildId: 'guild-1', memberId: '10001', permanent: undefined }])
  assert.ok(record.kickedAt instanceof Date)
})

test('member guard records failed join request approval attempts', async () => {
  const events: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async resolveJoinRequestDecision() {
        return {
          decision: 'approve',
          reason: 'verified_auto_approve',
          verificationState: 'verified',
          joinHandlingStrategy: 'join_request_review',
          autoApproveVerifiedJoin: true,
          autoApproveUnverifiedJoin: true,
        }
      },
      async recordJoinRequestEvent(input: unknown) {
        events.push(input)
      },
    },
    policyStore: policyStoreFor(['guild-1']),
    guardStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await assert.rejects(
    () => service.handleGuildMemberRequest(joinRequestSession({
      bot: {
        handleGuildMemberRequest: async () => {
          throw new Error('bot permission denied')
        },
      },
    })),
    /bot permission denied/,
  )

  assert.deepEqual(events, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    decision: 'approve',
    success: false,
    error: 'bot permission denied',
    rawEvent: { comment: '申请入群' },
  }])
})

test('member guard fail-closes when platform session creation is unavailable and syncs later', async () => {
  const savedRecords: any[] = []
  const updates: Array<{ id: string, input: Record<string, unknown> }> = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  const admissionEvents: unknown[] = []
  let backendAvailable = false
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        if (!backendAvailable) throw new Error('platform unavailable')
        return admissionResult('session-synced', 'token-synced')
      },
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() { return [] },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: any) { savedRecords.push(record) },
      async listBackendSyncPending() { return savedRecords.filter((record) => record.backendSyncPending) },
      async markBackendSynced(id: string, input: Record<string, unknown>) {
        updates.push({ id, input })
        const record = savedRecords.find((item) => item.id === id)
        Object.assign(record, input)
      },
      async markMuted() {},
      async markReminderSent() {},
      async markLastError(id: string, message: string) {
        const record = savedRecords.find((item) => item.id === id)
        record.lastError = message
      },
    },
    policyStore: {
      async resolvePolicy() {
        return {
          source: 'static',
          templateId: 'static',
          templateName: 'static',
          platform: 'qq',
          guildId: 'guild-1',
          joinHandlingStrategy: 'post_join_guard',
          muteDurationSeconds: 600,
          kickAfterMinutes: 30,
          reminderTemplate: '请等待机器人恢复认证链接。',
          exemptUsers: [],
        }
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'qq',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
        muteActions.push({ guildId, memberId, duration })
      },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
      },
    },
  } as any)

  assert.equal(savedRecords.length, 1)
  assert.equal(savedRecords[0].admissionSessionID, null)
  assert.equal(savedRecords[0].backendSyncPending, true)
  assert.match(savedRecords[0].lastError, /platform unavailable/)
  assert.deepEqual(muteActions, [{ guildId: 'guild-1', memberId: '10001', duration: 600_000 }])
  assert.match(sentMessages[0], /请等待机器人恢复认证链接/)
  assert.doesNotMatch(sentMessages[0], /admission\/a/)

  backendAvailable = true
  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    muteGuildMember: async () => {},
    sendMessage: async (_channelId: string, content: string) => {
      sentMessages.push(content)
      return ['message-1']
    },
  } as any])

  assert.equal(updates.length, 1)
  assert.equal(updates[0].id, savedRecords[0].id)
  assert.equal(updates[0].input.admissionSessionID, 'session-synced')
  assert.equal(updates[0].input.backendSyncPending, false)
  assert.match(sentMessages[1], /https:\/\/join\.stuhelper\.com\/verify\/token-synced/)
  assert.deepEqual(admissionEvents, [{
    sessionID: 'session-synced',
    input: {
      action: 'remind',
      success: true,
      messageID: 'message-1',
    },
  }])
})

test('member guard skips backend sync reminders when the local guard record is no longer active', async () => {
  const backendPendingRecord = {
    ...recordFor('session-local-pending'),
    admissionSessionID: null,
    backendSyncPending: true,
    lastError: 'platform unavailable',
  }
  const updates: Array<{ id: string, input: Record<string, unknown> }> = []
  const sentMessages: string[] = []
  const admissionEvents: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        return admissionResult('session-recovered', 'token-recovered')
      },
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() { return [] },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: {
      async listBackendSyncPending() { return [backendPendingRecord] },
      async markBackendSynced(id: string, input: Record<string, unknown>) {
        updates.push({ id, input })
        return false
      },
      async markReminderSent() {
        throw new Error('stale backend sync should not mark reminders')
      },
      async markLastError() {},
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async (_channelId: string, content: string) => {
      sentMessages.push(content)
      return ['message-stale']
    },
    muteGuildMember: async () => {},
    kickGuildMember: async () => {},
  } as any])

  assert.equal(updates.length, 1)
  assert.equal(updates[0].id, backendPendingRecord.id)
  assert.equal(updates[0].input.admissionSessionID, 'session-recovered')
  assert.deepEqual(sentMessages, [])
  assert.deepEqual(admissionEvents, [])
})

test('member guard kicks blacklisted members instead of pending backend sync', async () => {
  const savedRecords: unknown[] = []
  const kicks: Array<{ guildId: string, memberId: string, permanent?: boolean }> = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        throw new PlatformAPIError('member is blacklisted', 409, 'admission.member_blacklisted')
      },
    },
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: unknown) { savedRecords.push(record) },
      async markMuted() {},
      async markReminderSent() {},
    },
    policyStore: {
      async resolvePolicy() {
        return {
          source: 'static',
          templateId: 'static',
          joinHandlingStrategy: 'post_join_guard',
          exemptUsers: [],
        }
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'qq',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      kickGuildMember: async (guildId: string, memberId: string, permanent?: boolean) => {
        kicks.push({ guildId, memberId, permanent })
      },
      muteGuildMember: async () => {},
      sendMessage: async () => {},
    },
  } as any)

  assert.deepEqual(savedRecords, [])
  assert.deepEqual(kicks, [{ guildId: 'guild-1', memberId: '10001', permanent: false }])
})

test('member guard ignores duplicate join events for an active admission record', async () => {
  const savedRecords: any[] = []
  const createSessionCalls: unknown[] = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession(input: unknown) {
        createSessionCalls.push(input)
        return admissionResult('session-duplicate', 'token-duplicate')
      },
      async recordAdmissionEvent() {},
    },
    guardStore: {
      async findActiveBySubject() {
        return savedRecords[0] ?? null
      },
      async savePending(record: any) { savedRecords.push(record) },
      async markMuted() {},
      async markReminderSent() {},
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)
  const session = {
    platform: 'onebot',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
        muteActions.push({ guildId, memberId, duration })
      },
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        return ['message-duplicate']
      },
    },
  } as any

  await service.handleGuildMemberAdded(session)
  await service.handleGuildMemberAdded(session)

  assert.equal(createSessionCalls.length, 1)
  assert.equal(savedRecords.length, 1)
  assert.equal(muteActions.length, 1)
  assert.equal(sentMessages.length, 1)
  assert.equal(savedRecords[0].platform, 'qq')
  assert.match(sentMessages[0], /https:\/\/join\.stuhelper\.com\/verify\/token-duplicate/)
})

test('member guard retries backend reminder after initial group message send fails', async () => {
  const savedRecords: any[] = []
  const sentMessages: string[] = []
  const admissionEvents: unknown[] = []
  const reminderMarks: string[] = []
  const deduper = new AdmissionReminderDeduper()
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        return admissionResult('session-retry', 'token-retry')
      },
      async listPendingAdmissionActions() {
        return [
          action('session-retry', 'remind', {
            authURL: 'https://join.stuhelper.com/verify/token-retry',
          }),
        ]
      },
      async listPendingFreshmanForwards() { return [] },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        admissionEvents.push({ sessionID, input })
      },
    },
    guardStore: {
      async findActiveBySubject() { return null },
      async savePending(record: any) { savedRecords.push(record) },
      async markMuted() {},
      async markReminderSent(id: string, now: Date) {
        reminderMarks.push(`reminder:${id}`)
        const record = savedRecords.find((item) => item.id === id)
        if (record) record.reminderSentAt = now
      },
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID(sessionID: string) {
        return savedRecords.find((item) => item.admissionSessionID === sessionID) ?? null
      },
      async markLastError() {},
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    reminderDeduper: deduper,
  } as any)

  await assert.rejects(() => service.handleGuildMemberAdded({
    platform: 'onebot',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    bot: {
      muteGuildMember: async () => {},
      sendMessage: async (_channelId: string, content: string) => {
        sentMessages.push(content)
        throw new Error('send failed')
      },
    },
  } as any), /send failed/)

  assert.equal(savedRecords.length, 1)
  assert.equal(savedRecords[0].admissionSessionID, 'session-retry')
  assert.equal(sentMessages.length, 1)
  assert.deepEqual(reminderMarks, [])
  assert.deepEqual(admissionEvents, [])

  await service.scanPendingMembers([{
    platform: 'onebot',
    selfId: '514',
    sid: 'onebot:514',
    sendMessage: async (_channelId: string, content: string) => {
      sentMessages.push(content)
      return ['message-retry']
    },
    muteGuildMember: async () => {},
    kickGuildMember: async () => {},
  } as any])

  assert.equal(sentMessages.length, 2)
  assert.match(sentMessages[1], /https:\/\/join\.stuhelper\.com\/verify\/token-retry/)
  assert.deepEqual(reminderMarks, ['reminder:qq:514:guild-1:10001'])
  assert.deepEqual(admissionEvents, [{
    sessionID: 'session-retry',
    input: {
      action: 'remind',
      success: true,
      messageID: 'message-retry',
    },
  }])
})

test('member guard claims pending admission actions and reports results', async () => {
  const actions = [
    action('session-remind', 'remind', {
      authURL: 'https://join.stuhelper.com/verify/remind-token',
    }),
    action('session-release', 'release', { actionID: 'action-release', dispatchAttempt: 2 }),
    action('session-kick', 'kick'),
    action('session-blacklist', 'blacklist'),
  ] as const
  const claimCalls: unknown[] = []
  const events: unknown[] = []
  const actionEvents: unknown[] = []
  const messages: Array<{ channelId: string, content: string }> = []
  const mutes: Array<{ guildId: string, memberId: string, duration: number }> = []
  const kicks: Array<{ guildId: string, memberId: string, permanent?: boolean }> = []
  const marks: string[] = []
  const service = new MemberGuardService({
    platform: {
      async claimQueuedAdmissionActions(input: unknown) {
        claimCalls.push(input)
        return actions
      },
      async listPendingAdmissionActions() {
        throw new Error('fallback must claim queued admission actions before session-derived actions')
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        events.push({ sessionID, input })
      },
      async recordAdmissionActionEvent(actionID: string, input: unknown) {
        actionEvents.push({ actionID, input })
      },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID(sessionID: string) {
        return recordFor(sessionID)
      },
      async markReminderSent(id: string) { marks.push(`reminder:${id}`) },
      async markReleased(id: string) { marks.push(`released:${id}`) },
      async markKicked(id: string) { marks.push(`kicked:${id}`) },
      async markLastError() {},
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async (channelId: string, content: string) => {
      messages.push({ channelId, content })
      return ['message-1']
    },
    muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
      mutes.push({ guildId, memberId, duration })
    },
    kickGuildMember: async (guildId: string, memberId: string, permanent?: boolean) => {
      kicks.push({ guildId, memberId, permanent })
    },
  } as any])

  assert.deepEqual(claimCalls, [{ platform: 'qq', botSelfID: '514' }])
  assert.match(messages[0].content, /https:\/\/join\.stuhelper\.com\/verify\/remind-token/)
  assert.deepEqual(mutes, [{ guildId: 'guild-1', memberId: '10001', duration: 0 }])
  assert.deepEqual(kicks, [
    { guildId: 'guild-1', memberId: '10001', permanent: undefined },
    { guildId: 'guild-1', memberId: '10001', permanent: true },
  ])
  assert.deepEqual(marks, [
    'reminder:guard-session-remind',
    'released:guard-session-release',
    'kicked:guard-session-kick',
    'kicked:guard-session-blacklist',
  ])
  assert.deepEqual(events, [
    successEvent('session-remind', 'remind', 'message-1'),
    successEvent('session-kick', 'kick', 'message-1'),
    successEvent('session-blacklist', 'blacklist', 'message-1'),
  ])
  assert.deepEqual(actionEvents, [
    { actionID: 'action-release', input: { action: 'release', success: true, dispatchAttempt: 2 } },
  ])
})

test('member guard reports action failures, keeps errors visible, and continues the batch', async () => {
  const errors: string[] = []
  const events: unknown[] = []
  const marks: string[] = []
  const messages: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() {
        return [
          action('session-release', 'release'),
          action('session-remind', 'remind', {
            authURL: 'https://join.stuhelper.com/verify/remind-token',
          }),
        ]
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        events.push({ sessionID, input })
      },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID(sessionID: string) {
        return recordFor(sessionID)
      },
      async markLastError(_id: string, message: string) {
        errors.push(message)
      },
      async markReminderSent(id: string) { marks.push(`reminder:${id}`) },
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    muteGuildMember: async () => { throw new Error('mute failed') },
    sendMessage: async (_channelId: string, content: string) => {
      messages.push(content)
      return ['message-1']
    },
  } as any])

  assert.deepEqual(errors, ['mute failed'])
  assert.deepEqual(events, [{
    sessionID: 'session-release',
    input: {
      action: 'release',
      success: false,
      error: 'mute failed',
    },
  }, {
    sessionID: 'session-remind',
    input: {
      action: 'remind',
      success: true,
      messageID: 'message-1',
    },
  }])
  assert.deepEqual(marks, ['reminder:guard-session-remind'])
  assert.match(messages[0], /remind-token/)
})

test('member guard suppresses duplicate pending reminders shortly after a local reminder was sent', async () => {
  const events: unknown[] = []
  const marks: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() {
        return [
          action('session-remind', 'remind', {
            authURL: 'https://join.stuhelper.com/verify/remind-token',
          }),
        ]
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        events.push({ sessionID, input })
      },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID(sessionID: string) {
        return {
          ...recordFor(sessionID),
          reminderSentAt: new Date(),
        }
      },
      async markReminderSent(id: string) { marks.push(`reminder:${id}`) },
      async markLastError() { throw new Error('duplicate reminder should not fail the guard record') },
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => { throw new Error('duplicate reminder should not send another group message') },
  } as any])

  assert.deepEqual(events, [{
    sessionID: 'session-remind',
    input: {
      action: 'remind',
      success: true,
    },
  }])
  assert.deepEqual(marks, ['reminder:guard-session-remind'])
})

test('member guard suppresses scheduler reminder after admin command reminder even without local record', async () => {
  const events: unknown[] = []
  const deduper = new AdmissionReminderDeduper()
  deduper.remember('session-remind')
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() {
        return [
          action('session-remind', 'remind', {
            authURL: 'https://join.stuhelper.com/verify/remind-token',
          }),
        ]
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        events.push({ sessionID, input })
      },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID() { return null },
      async markLastError() { throw new Error('deduped reminder should not fail a missing guard record') },
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    reminderDeduper: deduper,
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => { throw new Error('deduped reminder should not send a group message') },
  } as any])

  assert.deepEqual(events, [{
    sessionID: 'session-remind',
    input: {
      action: 'remind',
      success: true,
    },
  }])
})

test('member guard suppresses stale reminder after admission was skipped locally', async () => {
  const events: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() {
        return [
          action('session-skipped', 'remind', {
            authURL: 'https://join.stuhelper.com/verify/skipped-token',
          }),
        ]
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        events.push({ sessionID, input })
      },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID() { return null },
      async findByAdmissionSessionID(sessionID: string) {
        return {
          ...recordFor(sessionID),
          releasedAt: new Date(),
          kickedAt: null,
        }
      },
      async markReminderSent() {
        throw new Error('stale skipped reminder should not mark reminders')
      },
      async markLastError() {
        throw new Error('stale skipped reminder should not fail the guard record')
      },
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => { throw new Error('stale skipped reminder should not send a group message') },
  } as any])

  assert.deepEqual(events, [{
    sessionID: 'session-skipped',
    input: {
      action: 'remind',
      success: true,
    },
  }])
})

test('member guard skips qq-only background polls without qq platform', async () => {
  let pendingActionCalls = 0
  let freshmanForwardCalls = 0
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() {
        pendingActionCalls += 1
        return []
      },
      async listPendingFreshmanForwards() {
        freshmanForwardCalls += 1
        return []
      },
    },
    guardStore: { async listBackendSyncPending() { return [] } },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{ selfId: '514', sid: 'missing:514' } as any])

  assert.equal(pendingActionCalls, 0)
  assert.equal(freshmanForwardCalls, 0)
})

test('member guard can explicitly disable freshman material forward polling', async () => {
  let freshmanForwardCalls = 0
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        freshmanForwardCalls += 1
        return []
      },
    },
    guardStore: { async listBackendSyncPending() { return [] } },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
    isFreshmanForwardEnabled: () => false,
  } as any)

  await service.scanPendingMembers([{
    platform: 'onebot',
    selfId: '514',
    sid: 'onebot:514',
    sendMessage: async () => ['message-1'],
  } as any])

  assert.equal(freshmanForwardCalls, 0)
})

test('member guard refuses pending admission actions outside local guard policy', async () => {
  const events: unknown[] = []
  const errors: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() {
        return [action('session-foreign', 'release', { guildID: 'guild-foreign' })]
      },
      async recordAdmissionEvent(sessionID: string, input: unknown) {
        events.push({ sessionID, input })
      },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
      async findActiveByAdmissionSessionID() { return null },
      async markLastError(_id: string, message: string) { errors.push(message) },
    },
    policyStore: policyStoreFor(['guild-1']),
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    muteGuildMember: async () => { throw new Error('mute should not be called') },
    sendMessage: async () => { throw new Error('send should not be called') },
  } as any])

  assert.deepEqual(errors, [])
  assert.deepEqual(events, [{
    sessionID: 'session-foreign',
    input: {
      action: 'release',
      success: false,
      error: 'admission action session-foreign targets unmanaged guild guild-foreign',
    },
  }])
})

function action(sessionID: string, actionName: string, overrides: Record<string, unknown> = {}) {
  return {
    sessionID,
    action: actionName,
    platform: 'qq',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    deadlineAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    ...overrides,
  }
}

function policyStoreFor(guildIds: readonly string[]) {
  return {
    async resolvePolicy(platform: string, guildId: string) {
      if (platform !== 'qq' || !guildIds.includes(guildId)) return null
      return {
        source: 'static',
        templateId: 'static',
        templateName: 'static',
        platform,
        guildId,
        joinHandlingStrategy: 'post_join_guard',
        muteDurationSeconds: 600,
        kickAfterMinutes: 30,
        reminderTemplate: '请先完成认证。',
        exemptUsers: [],
      }
    },
  }
}

function timeCodePolicyStore() {
  return {
    async resolvePolicy(platform: string, guildId: string) {
      if (platform !== 'qq' || guildId !== 'guild-1') return null
      return {
        source: 'static',
        templateId: 'static',
        templateName: 'static',
        platform,
        guildId,
        joinHandlingStrategy: 'post_join_time_code',
        muteDurationSeconds: 600,
        kickAfterMinutes: 30,
        reminderTemplate: '请先完成验证码。',
        exemptUsers: [],
      }
    },
  }
}

function recordFor(sessionID: string) {
  return {
    id: `guard-${sessionID}`,
    platform: 'qq',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    deadlineAt: new Date(Date.now() + 60 * 60 * 1000),
    admissionSessionID: sessionID,
    backendSyncPending: false,
  }
}

function timeCodeRecord(overrides: Record<string, unknown> = {}) {
  const now = new Date('2026-06-17T07:30:00.000Z')
  return {
    id: 'qq:514:guild-1:10001',
    platform: 'qq',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    memberName: 'Alice',
    verificationState: 'unbound',
    admissionSessionID: null,
    backendSyncPending: false,
    joinedAt: now,
    deadlineAt: new Date('2026-06-17T08:00:00.000Z'),
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: null,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  } as any
}

function activeRecordStore(savedRecords: any[]) {
  return {
    async findActiveBySubject() { return null },
    async savePending(record: any) { savedRecords.push(record) },
    async getActiveByID(id: string) {
      return savedRecords.find((record) => record.id === id && !record.releasedAt && !record.kickedAt)
    },
    async markMuted(id: string, now: Date) {
      const record = savedRecords.find((item) => item.id === id && !item.releasedAt && !item.kickedAt)
      if (!record) return false
      record.mutedAt = now
      return true
    },
    async markReminderSent(id: string, now: Date) {
      const record = savedRecords.find((item) => item.id === id && !item.releasedAt && !item.kickedAt)
      if (!record) return false
      record.reminderSentAt = now
      return true
    },
    async markReleased(id: string, now: Date) {
      const record = savedRecords.find((item) => item.id === id && !item.releasedAt && !item.kickedAt)
      if (!record) return false
      record.releasedAt = now
      return true
    },
  }
}

function memberAddedSession(overrides: Record<string, unknown> = {}) {
  return {
    platform: 'qq',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '10001',
    username: 'Alice',
    event: { user: { nick: 'Alice' } },
    ...overrides,
  } as any
}

function messageSession(overrides: Record<string, unknown> = {}) {
  return {
    platform: 'qq',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '10001',
    username: 'Alice',
    content: '',
    event: {
      guild: { id: 'guild-1' },
      user: { nick: 'Alice' },
    },
    bot: {
      sendMessage: async () => {},
    },
    ...overrides,
  } as any
}

function joinRequestSession(overrides: Record<string, unknown> = {}) {
  return {
    platform: 'onebot',
    selfId: '514',
    guildId: 'guild-1',
    channelId: 'guild-1',
    userId: '10001',
    messageId: 'request-1',
    event: {
      _data: { comment: '申请入群' },
      user: { nick: 'Alice' },
    },
    bot: {
      handleGuildMemberRequest: async () => {},
    },
    ...overrides,
  } as any
}

function successEvent(sessionID: string, actionName: string, messageID: string) {
  return {
    sessionID,
    input: {
      action: actionName,
      success: true,
      messageID,
    },
  }
}

function admissionResult(sessionID: string, token: string) {
  return {
    token,
    authURL: `https://join.stuhelper.com/verify/${token}`,
    session: {
      id: sessionID,
      platform: 'qq',
      guildID: 'guild-1',
      channelID: 'channel-1',
      qqID: '10001',
      status: 'joined_muted',
      tokenExpiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
      linkWaitDeadlineAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
      submissionWaitDeadlineAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
      initialMuteUntil: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
      projectionPending: false,
    },
  }
}
