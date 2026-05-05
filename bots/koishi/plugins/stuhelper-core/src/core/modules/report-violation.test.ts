import assert from 'node:assert/strict'
import test from 'node:test'

import { ViolationLevel } from './report-types'
import { handleReportViolation } from './report-violation'

test('moderation kick_blacklist writes a moderation_action blacklist entry', async () => {
  const host = createReportHost()

  const result = await handleReportViolation({
    host: host as never,
    session: createReportSession(host),
    userId: '10001',
    content: 'spam message',
    verbose: false,
    guildConfig: { autoProcess: true },
    violation: {
      level: ViolationLevel.HIGH,
      reason: 'spam',
      action: [{ type: 'kick_blacklist' }],
    },
  })

  assert.match(result, /踢出群聊并加入黑名单/)
  assert.deepEqual(host.kicks, [{ guildID: 'guild-1', userID: '10001', permanent: true }])
  assert.equal(host.createdBlacklists.length, 1)
  assert.equal(host.createdBlacklists[0].source, 'moderation_action')
  assert.equal(host.createdBlacklists[0].reasonCode, 'violation_review_blacklist')
  assert.equal(host.createdBlacklists[0].createdFrom, undefined)
  assert.equal(host.createdBlacklists[0].metadata.targetGuildID, 'guild-1')
  assert.match(String(host.createdBlacklists[0].metadata.reviewID), /^moderation:/)
  assert.match(String(host.createdBlacklists[0].metadata.workItemID), /^moderation:/)
})

function createReportHost() {
  const host = {
    kicks: [] as Array<{ guildID: string; userID: string; permanent?: boolean }>,
    createdBlacklists: [] as Array<Record<string, any>>,
    ctx: {
      logger() {
        return { info() {}, error() {}, warn() {} }
      },
      stuhelperGroupCenter: {
        pushMessage: async () => {},
      },
      $commander: {
        get() {
          return {
            execute: async () => 'legacy command path should not be used',
          }
        },
      },
    },
    memberBlacklistBackend: {
      async createMemberBlacklist(input: Record<string, any>) {
        host.createdBlacklists.push(input)
        return { id: 'blacklist-1', ...input }
      },
    },
    logCommand: async () => {},
  }
  return host
}

function createReportSession(host: ReturnType<typeof createReportHost>) {
  return {
    platform: 'qq',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: '90001',
    user: { authority: 1 },
    bot: {
      kickGuildMember: async (guildID: string, userID: string, permanent?: boolean) => {
        host.kicks.push({ guildID, userID, permanent })
      },
    },
  }
}
