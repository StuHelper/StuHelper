import assert from 'node:assert/strict'
import test from 'node:test'

import { PlatformAPIError } from '@stuhelper/koishi-shared'

import { MemberGuardService } from './member-guard'

test('member guard creates admission session, mutes, and sends canonical auth link', async () => {
  const savedRecords: unknown[] = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  const createSessionCalls: unknown[] = []
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession(input: unknown) {
        createSessionCalls.push(input)
        return {
          token: 'token-1',
          authURL: 'https://auth.stuhelper.com/admission/a/token-1?qq=10001',
          session: {
            id: 'session-1',
            platform: 'mock',
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
    },
    guardStore: {
      async savePending(record: unknown) { savedRecords.push(record) },
      async markMuted() {},
      async markReminderSent() {},
    },
    policyStore: {
      async resolvePolicy() {
        return {
          source: 'static',
          templateId: 'static',
          exemptUsers: [],
        }
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'mock',
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

  assert.deepEqual(createSessionCalls, [{
    platform: 'mock',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    qqNickname: 'Alice',
    botSelfID: '514',
  }])
  assert.equal(muteActions.length, 1)
  assert.equal(muteActions[0].guildId, 'guild-1')
  assert.equal(muteActions[0].memberId, '10001')
  assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
  assert.equal(savedRecords.length, 1)
  assert.match(JSON.stringify(savedRecords[0]), /session-1/)
  assert.match(sentMessages[0], /https:\/\/auth\.stuhelper\.com\/admission\/a\/token-1\?qq=10001/)
  assert.doesNotMatch(sentMessages[0], /buaa\.team|sso\.stuhelper\.com/)
})

test('member guard fail-closes when platform session creation is unavailable and syncs later', async () => {
  const savedRecords: any[] = []
  const updates: Array<{ id: string, input: Record<string, unknown> }> = []
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []
  let backendAvailable = false
  const service = new MemberGuardService({
    platform: {
      async createAdmissionSession() {
        if (!backendAvailable) throw new Error('platform unavailable')
        return admissionResult('session-synced', 'token-synced')
      },
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() { return [] },
    },
    guardStore: {
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
          platform: 'mock',
          guildId: 'guild-1',
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
    platform: 'mock',
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
    platform: 'mock',
    selfId: '514',
    sid: 'mock:514',
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
  assert.match(sentMessages[1], /https:\/\/auth\.stuhelper\.com\/admission\/a\/token-synced\?qq=10001/)
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
      async savePending(record: unknown) { savedRecords.push(record) },
      async markMuted() {},
      async markReminderSent() {},
    },
    policyStore: {
      async resolvePolicy() {
        return {
          source: 'static',
          templateId: 'static',
          exemptUsers: [],
        }
      },
    },
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.handleGuildMemberAdded({
    platform: 'mock',
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

test('member guard executes pending admission actions and reports results', async () => {
  const actions = [
    action('session-remind', 'remind', {
      authURL: 'https://auth.stuhelper.com/admission/a/remind-token?qq=10001',
    }),
    action('session-release', 'release'),
    action('session-kick', 'kick'),
    action('session-blacklist', 'blacklist'),
  ] as const
  const listCalls: unknown[] = []
  const events: unknown[] = []
  const messages: Array<{ channelId: string, content: string }> = []
  const mutes: Array<{ guildId: string, memberId: string, duration: number }> = []
  const kicks: Array<{ guildId: string, memberId: string, permanent?: boolean }> = []
  const marks: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions(input: unknown) {
        listCalls.push(input)
        return actions
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
      async markReminderSent(id: string) { marks.push(`reminder:${id}`) },
      async markReleased(id: string) { marks.push(`released:${id}`) },
      async markKicked(id: string) { marks.push(`kicked:${id}`) },
      async markLastError() {},
    },
    policyStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'mock',
    selfId: '514',
    sid: 'mock:514',
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

  assert.deepEqual(listCalls, [{ platform: 'mock', botSelfID: '514' }])
  assert.match(messages[0].content, /https:\/\/auth\.stuhelper\.com\/admission\/a\/remind-token\?qq=10001/)
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
    successEvent('session-release', 'release', 'message-1'),
    successEvent('session-kick', 'kick', 'message-1'),
    successEvent('session-blacklist', 'blacklist', 'message-1'),
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
            authURL: 'https://auth.stuhelper.com/admission/a/remind-token?qq=10001',
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
    policyStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'mock',
    selfId: '514',
    sid: 'mock:514',
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

function action(sessionID: string, actionName: string, overrides: Record<string, unknown> = {}) {
  return {
    sessionID,
    action: actionName,
    platform: 'mock',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    deadlineAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    ...overrides,
  }
}

function recordFor(sessionID: string) {
  return {
    id: `guard-${sessionID}`,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    deadlineAt: new Date(Date.now() + 60 * 60 * 1000),
    admissionSessionID: sessionID,
    backendSyncPending: false,
  }
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
    authURL: `https://auth.stuhelper.com/admission/a/${token}?qq=10001`,
    session: {
      id: sessionID,
      platform: 'mock',
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
