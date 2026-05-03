import assert from 'node:assert/strict'
import test from 'node:test'

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
