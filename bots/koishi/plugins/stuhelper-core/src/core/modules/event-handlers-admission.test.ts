import assert from 'node:assert/strict'
import test from 'node:test'

import { registerEventListeners } from './event-handlers'

test('guild-member-request keeps one listener and approves through admission after legacy rules', async () => {
  const host = createEventHost()
  registerEventListeners(host as any)

  assert.equal(host.handlers.filter((item) => item.event === 'guild-member-request').length, 1)
  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{ requestID: 'req-1', approve: true, reason: undefined }])
  assert.deepEqual(host.joinEvents, [{
    platform: 'mock',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
    success: true,
    rawEvent: { comment: '我是新生' },
  }])
})

test('guild-member-request preserves blacklist and keyword ordering before admission', async () => {
  const host = createEventHost()
  host.data.blacklist.rows = { '10001': true }
  registerEventListeners(host as any)

  await host.emitGuildMemberRequest(createRequestSession(host, { comment: '我是新生' }))

  assert.deepEqual(host.approvals, [{ requestID: 'req-1', approve: false, reason: '您在黑名单中' }])
  assert.equal(host.admissionAccessChecks, 0)

  const keywordHost = createEventHost()
  keywordHost.config.keywords = ['校内群']
  registerEventListeners(keywordHost as any)
  await keywordHost.emitGuildMemberRequest(createRequestSession(keywordHost, { comment: '申请加入校内群' }))

  assert.deepEqual(keywordHost.approvals, [{ requestID: 'req-1', approve: true, reason: undefined }])
  assert.equal(keywordHost.admissionAccessChecks, 0)
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
    platform: 'mock',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'req-1',
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
    admissionAccessChecks: 0,
    approveError: null as Error | null,
    ctx: {
      on(event: string, handler: (session: unknown) => unknown) {
        host.handlers.push({ event, handler })
      },
    },
    data: {
      blacklist: { rows: {}, getAll() { return this.rows } },
      leaveRecords: { getAll() { return {} } },
      groupConfig: { getAll() { return {} } },
    },
    config: {
      keywords: [] as string[],
      friendRequest: { enabled: false, keywords: [], rejectMessage: '' },
      guildRequest: { enabled: false, rejectMessage: '' },
    },
    admissionPlatform: {
      async getAdmissionQQAccess(qqID: string) {
        assert.equal(qqID, '10001')
        host.admissionAccessChecks++
        return { canJoin: true, autoApproveJoin: true }
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
    platform: 'mock',
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
