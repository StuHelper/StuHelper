import assert from 'node:assert/strict'
import test from 'node:test'

import type { ReportModule } from './report.module'
import type { ReportActionSession } from './report-session'
import { handleReportViolation } from './report-violation.ts'
import { ViolationLevel } from './report-types.ts'

function createStore<T>() {
  const values = new Map<string, T>()
  return {
    get: (key: string) => values.get(key),
    set: (key: string, value: T) => values.set(key, value),
    flush: () => undefined,
    values,
  }
}

function createHost() {
  const logs: unknown[] = []
  const pushed: unknown[] = []
  const createdBlacklists: unknown[] = []
  return {
    memberBlacklistBackend: {
      async createMemberBlacklist(input: unknown) {
        createdBlacklists.push(input)
        return { id: 'entry-1' }
      },
    },
    config: {
      report: { autoProcess: true },
    },
    data: {
      warns: createStore<Record<string, { count: number; timestamp: number }>>(),
      mutes: createStore<Record<string, { startTime: number; duration: number; remainingTime: number }>>(),
    },
    ctx: {
      stuhelperGroupCenter: {
        pushMessage: async (...args: unknown[]) => pushed.push(args),
      },
    },
    logCommand: async (...args: unknown[]) => logs.push(args),
    logs,
    pushed,
    createdBlacklists,
  } as unknown as ReportModule & {
    readonly logs: unknown[]
    readonly pushed: unknown[]
    readonly createdBlacklists: unknown[]
  }
}

test('handleReportViolation performs ban and warn directly without mutating session authority', async () => {
  const host = createHost()
  const muted: unknown[] = []
  const sessionUser = { authority: 1, permissions: ['report'] }
  const session = {
    platform: 'qq',
    guildId: 'guild-1',
    userId: 'operator-qq',
    user: sessionUser,
    execute: async () => {
      throw new Error('session.execute should not be used for auto moderation')
    },
    bot: {
      muteGuildMember: async (...args: unknown[]) => muted.push(args),
    },
  } as unknown as ReportActionSession

  const result = await handleReportViolation({
    host,
    session,
    userId: 'target-qq',
    content: 'bad content',
    verbose: false,
    guildConfig: { autoProcess: true },
    violation: {
      level: ViolationLevel.HIGH,
      reason: 'abuse',
      action: [{ type: 'ban', time: 30 }, { type: 'warn', count: 2 }],
    },
  })

  assert.equal(result, '已对用户 target-qq 执行：禁言30秒、警告2次，严重违规。')
  assert.deepEqual(muted, [['guild-1', 'target-qq', 30_000]])
  assert.equal(session.user, sessionUser)
  assert.deepEqual(session.user, { authority: 1, permissions: ['report'] })
  assert.equal(host.data.warns.values.get('guild-1')?.['target-qq'].count, 2)
  assert.equal(host.data.mutes.values.get('guild-1')?.['target-qq'].duration, 30_000)
})

test('handleReportViolation creates backend member blacklist for kick_blacklist action', async () => {
  const host = createHost()
  const kicked: unknown[] = []
  const session = {
    platform: 'qq',
    guildId: 'guild-1',
    userId: 'operator-qq',
    content: 'reported message',
    messageId: 'message-1',
    bot: {
      kickGuildMember: async (...args: unknown[]) => kicked.push(args),
    },
  } as unknown as ReportActionSession

  const result = await handleReportViolation({
    host,
    session,
    userId: 'target-qq',
    content: 'reported message',
    verbose: false,
    guildConfig: { autoProcess: true },
    violation: {
      level: ViolationLevel.CRITICAL,
      reason: 'abuse',
      action: [{ type: 'kick_blacklist' }],
    },
  })

  assert.equal(result, '已对用户 target-qq 执行：踢出群聊并加入黑名单，极其严重违规。')
  assert.deepEqual(kicked, [['guild-1', 'target-qq', true]])
  assert.deepEqual(host.createdBlacklists, [{
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: 'target-qq',
    scopeType: 'guild',
    guildID: 'guild-1',
    source: 'moderation_action',
    reasonCode: 'violation_review_blacklist',
    reasonText: 'moderation review blacklist: abuse',
    createdFrom: 'moderation_review',
    metadata: {
      reviewID: 'moderation:qq:guild-1:message-1:target-qq',
      workItemID: 'moderation:qq:guild-1:message-1:target-qq',
      targetGuildID: 'guild-1',
    },
  }])
})

test('handleReportViolation reports non-Error action failures without throwing again', async () => {
  const host = createHost()
  const session = {
    platform: 'qq',
    guildId: 'guild-1',
    userId: 'operator-qq',
    bot: {
      kickGuildMember: async () => {
        throw 'adapter refused kick'
      },
    },
  } as unknown as ReportActionSession

  const result = await handleReportViolation({
    host,
    session,
    userId: 'target-qq',
    content: 'reported message',
    verbose: false,
    guildConfig: { autoProcess: true },
    violation: {
      level: ViolationLevel.HIGH,
      reason: 'abuse',
      action: [{ type: 'kick' }],
    },
  })

  assert.equal(result, '已对用户 target-qq 执行：kick操作失败，严重违规。')
  assert.deepEqual(host.logs.at(-1), [{
    session,
    command: 'report-handle',
    target: 'target-qq',
    details: '严重违规，处理: 踢出群聊，内容: reported message',
  }])
})
