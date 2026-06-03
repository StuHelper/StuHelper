import assert from 'node:assert/strict'
import test from 'node:test'
import { PlatformAPIError } from '@stuhelper/koishi-shared'

import { registerEventListeners } from './event-handlers'

test('guild-member-request keeps one listener and approves through admission before legacy keyword rules', async () => {
  const host = createEventHost()
  host.config.keywords = ['新生']
  registerEventListeners(host as any)

  assert.equal(host.handlers.filter((item) => item.event === 'guild-member-request').length, 1)
  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{ requestID: 'req-1', approve: true, reason: undefined }])
  assert.deepEqual(host.joinEvents, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    decision: 'approve',
    success: true,
    rawEvent: { comment: '我是新生' },
  }])
  assert.deepEqual(host.joinDecisionRequests, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    rawEvent: { comment: '我是新生' },
  }])
})

test('guild-member-request falls back to legacy keyword approval when admission policy is missing', async () => {
  const host = createEventHost()
  host.config.keywords = ['新生']
  host.joinDecisionError = new PlatformAPIError('admission policy not found', 404, 'admission.policy_not_found')
  registerEventListeners(host as any)

  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{ requestID: 'req-1', approve: true, reason: undefined }])
  assert.equal(host.joinEvents.length, 0)
  assert.deepEqual(host.joinDecisionRequests, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    rawEvent: { comment: '我是新生' },
  }])
})

test('guild-member-request rejects backend member blacklist before local approval rules', async () => {
  const host = createEventHost()
  host.memberBlacklistBlocked = true
  registerEventListeners(host as any)

  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{ requestID: 'req-1', approve: false, reason: '您在黑名单中' }])
  assert.equal(host.memberBlacklistAccessChecks, 1)
  assert.equal(host.joinEvents.length, 0)

  const keywordHost = createEventHost()
  keywordHost.config.keywords = ['校内群']
  registerEventListeners(keywordHost as any)
  await keywordHost.emitGuildMemberRequest(createRequestSession(keywordHost, { comment: '申请加入校内群' }))

  assert.deepEqual(keywordHost.approvals, [{ requestID: 'req-1', approve: true, reason: undefined }])
  assert.equal(keywordHost.memberBlacklistAccessChecks, 1)
})

test('guild-member-request keeps blacklist backend outages visible and falls through', async () => {
  const host = createEventHost()
  host.memberBlacklistAccessError = new Error('backend timeout')
  registerEventListeners(host as any)

  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{ requestID: 'req-1', approve: true, reason: undefined }])
  assert.deepEqual(host.joinEvents, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    decision: 'approve',
    success: true,
    rawEvent: { comment: '我是新生' },
  }])
})

test('guild-member-request rejects when admission policy disables auto approval', async () => {
  const host = createEventHost()
  host.joinDecision = {
    decision: 'reject',
    reason: 'unverified_auto_approve_disabled',
    verificationState: 'unverified',
    autoApproveVerifiedJoin: true,
    autoApproveUnverifiedJoin: false,
  }
  registerEventListeners(host as any)

  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{
    requestID: 'req-1',
    approve: false,
    reason: 'unverified_auto_approve_disabled',
  }])
  assert.deepEqual(host.joinEvents, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    decision: 'reject',
    success: true,
    rawEvent: { comment: '我是新生' },
  }])
})

test('guild-member-request reports admission approve failures and keeps error visible', async () => {
  const host = createEventHost()
  host.approveError = new Error('bot permission denied')
  registerEventListeners(host as any)

  await assert.rejects(
    host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' })),
    /bot permission denied/,
  )

  assert.deepEqual(host.joinEvents, [{
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    decision: 'approve',
    success: false,
    error: 'bot permission denied',
    rawEvent: { comment: '我是新生' },
  }])
})

function createEventHost() {
  const host = {
    handlers: [] as Array<{ event: string, handler: (session: unknown) => unknown }>,
    approvals: [] as Array<{ requestID: string, approve: boolean, reason?: string }>,
    joinEvents: [] as unknown[],
    joinDecisionRequests: [] as unknown[],
    joinDecision: {
      decision: 'approve',
      reason: 'unverified_auto_approve',
      verificationState: 'unverified',
      autoApproveVerifiedJoin: true,
      autoApproveUnverifiedJoin: true,
    },
    memberBlacklistAccessChecks: 0,
    memberBlacklistBlocked: false,
    memberBlacklistAccessError: null as Error | null,
    joinDecisionError: null as Error | null,
    approveError: null as Error | null,
    ctx: {
      on(event: string, handler: (session: unknown) => unknown) {
        host.handlers.push({ event, handler })
      },
    },
    data: {
      leaveRecords: { getAll() { return {} } },
      groupConfig: { getAll() { return {} } },
    },
    config: {
      keywords: [] as string[],
      friendRequest: { enabled: false, keywords: [], rejectMessage: '' },
      guildRequest: { enabled: false, rejectMessage: '' },
    },
    admissionPlatform: {
      async getMemberBlacklistAccess(input: { platform: string; subjectID: string; guildID?: string }) {
        assert.equal(input.platform, 'qq')
        assert.equal(input.subjectID, '10001')
        assert.equal(input.guildID, 'guild-1')
        host.memberBlacklistAccessChecks++
        if (host.memberBlacklistAccessError) throw host.memberBlacklistAccessError
        return host.memberBlacklistBlocked
          ? { canJoin: false, decision: 'blocked', reason: 'member_blacklisted' }
          : { canJoin: true, decision: 'allowed' }
      },
      async resolveJoinRequestDecision(input: unknown) {
        host.joinDecisionRequests.push(input)
        if (host.joinDecisionError) throw host.joinDecisionError
        return host.joinDecision
      },
      async recordJoinRequestEvent(input: unknown) {
        host.joinEvents.push(input)
      },
    },
    async emitGuildMemberRequest(session: unknown) {
      const handler = host.handlers.find((item) => item.event === 'guild-member-request')?.handler
      assert.ok(handler)
      await handler(session)
    },
  }
  return host
}

function createRequestSession(host: ReturnType<typeof createEventHost>, input: { comment: string }) {
  return {
    platform: 'onebot',
    guildId: 'guild-1',
    userId: '10001',
    messageId: 'req-1',
    content: input.comment,
    event: { _data: { comment: input.comment } },
    bot: {
      handleGuildMemberRequest: async (requestID: string, approve: boolean, reason?: string) => {
        if (host.approveError) {
          throw host.approveError
        }
        host.approvals.push({ requestID, approve, reason })
      },
    },
  }
}
