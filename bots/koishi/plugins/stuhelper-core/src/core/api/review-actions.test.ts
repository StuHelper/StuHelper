import assert from 'node:assert/strict'
import test from 'node:test'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'
import {
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
} from '@stuhelper/koishi-moderation-core'

import {
  handleWorkItemAction,
  normalizeLegacyReviewAction,
  parseWorkItemActionInput,
} from './review-actions'

test('handleWorkItemAction approves guarded admission members by releasing the mute and marking record released', async () => {
  const now = new Date('2026-04-21T08:00:00.000Z')
  const guardRecord = createGuardRecord()
  const deps = createActionDeps({
    guardRecords: [guardRecord],
    reports: [],
    now,
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'admission',
    itemId: guardRecord.id,
    action: 'approve',
    note: '已人工核验',
  }, createActor())

  assert.equal(message, `已放行待准入成员：${guardRecord.memberId}`)
  assert.deepEqual(deps.botCalls.mute, [[guardRecord.guildId, guardRecord.memberId, 0]])
  assert.equal(deps.databaseState.guardUpdates[0].query.id, guardRecord.id)
  assert.ok(deps.databaseState.guardUpdates.at(-1)?.patch.releasedAt instanceof Date)
  assert.equal(deps.events[0].type, 'join_released')
})

test('handleWorkItemAction creates review from report and removes report from open queue', async () => {
  const report = createReportRecord()
  const deps = createActionDeps({
    guardRecords: [],
    reports: [report],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'report',
    itemId: report.id,
    action: 'create-review',
    note: '转人工复核',
  }, createActor())

  assert.equal(message, `已把举报转成复核：${report.targetMemberId}`)
  assert.equal(deps.createdReviews.length, 1)
  assert.equal(deps.createdReviews[0].actionType, 'kick_and_block')
  assert.deepEqual(deps.databaseState.removes, [{ table: MODERATION_REPORT_TABLE, query: { id: report.id } }])
  assert.deepEqual(deps.operationLog, [
    'create-review:rp-1',
    `remove:${MODERATION_REPORT_TABLE}:rp-1`,
    'append-event:review_created',
  ])
})

test('handleWorkItemAction rejects pending reviews without executing moderation action', async () => {
  const review = createReviewRecord()
  const deps = createActionDeps({
    guardRecords: [],
    reports: [],
    reviews: [review],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'review',
    itemId: review.id,
    action: 'reject',
    note: '证据不足',
  }, createActor())

  assert.equal(message, `已驳回复核：${review.memberId}`)
  assert.deepEqual(deps.databaseState.reviewUpdates, [{
    id: review.id,
    status: 'rejected',
    operatorMemberId: 'admin-42',
    resolutionNote: '证据不足',
  }])
  assert.equal(deps.botCalls.kick.length, 0)
})

test('handleWorkItemAction executes pending review items with the current operator identity and permanent flag', async () => {
  const review = createReviewRecord({ actionType: 'kick_and_block' })
  const deps = createActionDeps({
    guardRecords: [createGuardRecord({ memberId: review.memberId })],
    reports: [],
    reviews: [review],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'review',
    itemId: review.id,
    action: 'execute',
    note: '证据充分',
  }, createActor())

  assert.equal(message, `已执行复核动作：${review.memberId}`)
  assert.deepEqual(deps.actionCalls.kick, [{
    guildId: review.guildId,
    channelId: review.channelId,
    memberId: review.memberId,
    permanent: true,
    reason: review.reason,
  }])
  assert.deepEqual(deps.databaseState.reviewUpdates.at(-1), {
    id: review.id,
    status: 'executed',
    operatorMemberId: 'admin-42',
    resolutionNote: '证据充分',
  })
  assert.equal(deps.events[0].payload.operatorMemberId, 'admin-42')
})

test('handleWorkItemAction defers guarded admission members by extending the deadline window', async () => {
  const guardRecord = createGuardRecord()
  const now = new Date('2026-04-21T08:00:00.000Z')
  const deps = createActionDeps({
    guardRecords: [guardRecord],
    reports: [],
    reviews: [],
    now,
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'admission',
    itemId: guardRecord.id,
    action: 'defer',
    note: '等待线下核验',
  }, createActor())

  assert.equal(message, `已延期待准入成员：${guardRecord.memberId}`)
  assert.equal(
    String(deps.databaseState.guardUpdates[0].patch.deadlineAt),
    new Date('2026-04-21T08:30:00.000Z').toString(),
  )
})

test('handleWorkItemAction denies guarded admission members by kicking the member and recording the operator action', async () => {
  const guardRecord = createGuardRecord()
  const now = new Date('2026-04-21T08:00:00.000Z')
  const deps = createActionDeps({
    guardRecords: [guardRecord],
    reports: [],
    reviews: [],
    now,
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'admission',
    itemId: guardRecord.id,
    action: 'deny',
    note: '身份异常',
  }, createActor())

  assert.equal(message, `已拒绝待准入成员：${guardRecord.memberId}`)
  assert.deepEqual(deps.botCalls.kick, [[guardRecord.guildId, guardRecord.memberId]])
  assert.match(deps.botCalls.send[0][1], /已被人工拒绝准入/)
  assert.ok(deps.databaseState.guardUpdates.at(-1)?.patch.kickedAt instanceof Date)
  assert.equal(deps.events[0].payload.operatorMemberId, 'admin-42')
  assert.equal(deps.events[0].payload.action, 'deny-admission')
})

test('handleWorkItemAction dismisses report work items from the open queue', async () => {
  const report = createReportRecord()
  const deps = createActionDeps({
    guardRecords: [],
    reports: [report],
    reviews: [],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'report',
    itemId: report.id,
    action: 'dismiss',
    note: '证据不足',
  }, createActor())

  assert.equal(message, `已驳回举报：${report.targetMemberId}`)
  assert.deepEqual(deps.databaseState.removes, [{ table: MODERATION_REPORT_TABLE, query: { id: report.id } }])
  assert.deepEqual(deps.operationLog, [
    `remove:${MODERATION_REPORT_TABLE}:rp-1`,
    'append-event:action_executed',
  ])
})

test('handleWorkItemAction escalates reports before appending the audit event', async () => {
  const report = createReportRecord()
  const deps = createActionDeps({
    guardRecords: [],
    reports: [report],
    reviews: [],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  const message = await handleWorkItemAction(deps, {
    kind: 'report',
    itemId: report.id,
    action: 'escalate',
    note: '升级为高风险',
  }, createActor())

  assert.equal(message, `已升级举报：${report.targetMemberId}`)
  assert.deepEqual(deps.updatedReports, [{
    id: report.id,
    aiStatus: 'completed',
    aiSeverity: 'high',
    aiSummary: '升级为高风险',
  }])
  assert.deepEqual(deps.operationLog, [
    'update-report:rp-1',
    'append-event:action_executed',
  ])
})

test('handleWorkItemAction rejects already resolved review items', async () => {
  const review = createReviewRecord({ status: 'executed' })
  const deps = createActionDeps({
    guardRecords: [],
    reports: [],
    reviews: [review],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  await assert.rejects(
    () => handleWorkItemAction(deps, {
      kind: 'review',
      itemId: review.id,
      action: 'reject',
    }, createActor()),
    /review is already resolved/,
  )
})

test('handleWorkItemAction surfaces missing records and missing managed bots', async () => {
  const review = createReviewRecord()
  const depsWithoutReview = createActionDeps({
    guardRecords: [],
    reports: [],
    reviews: [],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })
  const depsWithoutBot = createActionDeps({
    guardRecords: [],
    reports: [],
    reviews: [review],
    bots: [],
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  await assert.rejects(
    () => handleWorkItemAction(depsWithoutReview, {
      kind: 'review',
      itemId: review.id,
      action: 'execute',
    }, createActor()),
    /review not found/,
  )
  await assert.rejects(
    () => handleWorkItemAction(depsWithoutBot, {
      kind: 'review',
      itemId: review.id,
      action: 'execute',
    }, createActor()),
    /console bot not found/,
  )
})

test('handleWorkItemAction preserves the original review execution error when rollback also fails', async () => {
  const review = createReviewRecord()
  const deps = createActionDeps({
    guardRecords: [],
    reports: [],
    reviews: [review],
    kickError: new Error('kick failed'),
    reviewRollbackError: new Error('database unavailable'),
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  await assert.rejects(
    () => handleWorkItemAction(deps, {
      kind: 'review',
      itemId: review.id,
      action: 'execute',
    }, createActor()),
    /kick failed/,
  )
  assert.match(deps.logErrors[0], /rollback failed/)
})

test('handleWorkItemAction preserves the original admission denial error when rollback also fails', async () => {
  const guardRecord = createGuardRecord()
  const deps = createActionDeps({
    guardRecords: [guardRecord],
    reports: [],
    reviews: [],
    guardKickError: new Error('kick admission failed'),
    guardRollbackError: new Error('guard rollback unavailable'),
    now: new Date('2026-04-21T08:00:00.000Z'),
  })

  await assert.rejects(
    () => handleWorkItemAction(deps, {
      kind: 'admission',
      itemId: guardRecord.id,
      action: 'deny',
    }, createActor()),
    /kick admission failed/,
  )
  assert.match(deps.logErrors[0], /rollback failed/)
})

test('review action parsers reject malformed payloads', () => {
  assert.throws(
    () => normalizeLegacyReviewAction(null),
    /review action input must be an object/,
  )
  assert.throws(
    () => normalizeLegacyReviewAction({
      reviewId: 'rv-1',
      action: 'approve',
    }),
    /action must be one of: execute, reject/,
  )
  assert.throws(
    () => parseWorkItemActionInput({
      kind: 'appeal',
      itemId: 'rv-1',
      action: 'execute',
    }),
    /kind must be one of: review, admission, report/,
  )
  assert.throws(
    () => parseWorkItemActionInput({
      kind: 'report',
      itemId: ' ',
      action: 'dismiss',
    }),
    /itemId must be a non-empty string/,
  )
  assert.throws(
    () => parseWorkItemActionInput({
      kind: 'admission',
      itemId: 'gm-1',
      action: 'execute',
    }),
    /action must be one of: approve, deny, defer/,
  )
})

function createActionDeps(input: {
  guardRecords: any[]
  reports: any[]
  reviews?: any[]
  bots?: any[]
  kickError?: Error
  reviewRollbackError?: Error
  guardRollbackError?: Error
  guardKickError?: Error
  now: Date
}) {
  const guardUpdates: Array<{ query: Record<string, unknown>; patch: Record<string, unknown> }> = []
  const reviewUpdates: Array<{ id: string; status: string; operatorMemberId: string | null; resolutionNote: string | null }> = []
  const removes: Array<{ table: string; query: Record<string, unknown> }> = []
  const events: any[] = []
  const createdReviews: any[] = []
  const updatedReports: Array<{ id: string; aiStatus: string; aiSeverity: string; aiSummary: string | null }> = []
  const operationLog: string[] = []
  const actionCalls = {
    kick: [] as Array<{ guildId: string; channelId: string; memberId: string; permanent: boolean; reason: string }>,
  }
  const botCalls = {
    mute: [] as Array<[string, string, number]>,
    kick: [] as Array<[string, string]>,
    send: [] as Array<[string, string]>,
  }
  const logErrors: string[] = []
  const defaultBots = [{
    platform: 'onebot',
    selfId: 'bot',
    muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
      botCalls.mute.push([guildId, memberId, duration])
    },
    kickGuildMember: async (guildId: string, memberId: string) => {
      if (input.guardKickError) {
        throw input.guardKickError
      }
      botCalls.kick.push([guildId, memberId])
    },
    sendMessage: async (channelId: string, content: string) => {
      botCalls.send.push([channelId, content])
    },
  }]
  const bots = input.bots ?? defaultBots

  return {
    ctx: {
      bots,
      logger: () => ({
        error: (...args: unknown[]) => {
          logErrors.push(args.map((item) => String(item)).join(' '))
        },
      }),
      database: {
        get: async (table: string, query: Record<string, unknown>) => {
          if (table === GUARD_MEMBER_TABLE) {
            return input.guardRecords.filter((record) => record.id === query.id)
          }
          if (table === MODERATION_REVIEW_TABLE) {
            return (input.reviews || []).filter((record) => record.id === query.id)
          }
          if (table === MODERATION_REPORT_TABLE) {
            return input.reports.filter((record) => record.id === query.id)
          }
          return []
        },
        set: async (table: string, query: Record<string, unknown>, patch: Record<string, unknown>) => {
          if (
            table === MODERATION_REVIEW_TABLE
            && query.status === 'approved'
            && patch.status === 'pending'
            && input.reviewRollbackError
          ) {
            throw input.reviewRollbackError
          }
          if (
            table === GUARD_MEMBER_TABLE
            && 'lastError' in patch
            && !('releasedAt' in patch)
            && !('kickedAt' in patch)
            && input.guardRollbackError
          ) {
            throw input.guardRollbackError
          }
          if (table === GUARD_MEMBER_TABLE) {
            guardUpdates.push({ query, patch })
          }
          if (table === MODERATION_REVIEW_TABLE) {
            reviewUpdates.push({
              id: String(query.id),
              status: String(patch.status),
              operatorMemberId: (patch.operatorMemberId as string | null) ?? null,
              resolutionNote: (patch.resolutionNote as string | null) ?? null,
            })
          }
          return { matched: 1, modified: 1 }
        },
        remove: async (table: string, query: Record<string, unknown>) => {
          operationLog.push(`remove:${table}:${String(query.id)}`)
          removes.push({ table, query })
          return { removed: 1 }
        },
        transact: async <T>(callback: (database: any) => Promise<T>) => {
          return callback({
            get: async (table: string, query: Record<string, unknown>) => {
              if (table === GUARD_MEMBER_TABLE) {
                return input.guardRecords.filter((record) => record.id === query.id)
              }
              if (table === MODERATION_REVIEW_TABLE) {
                return (input.reviews || []).filter((record) => record.id === query.id)
              }
              if (table === MODERATION_REPORT_TABLE) {
                return input.reports.filter((record) => record.id === query.id)
              }
              return []
            },
            set: async (_table: string, _query: Record<string, unknown>, _patch: Record<string, unknown>) => ({ matched: 1, modified: 1 }),
            remove: async (table: string, query: Record<string, unknown>) => {
              operationLog.push(`remove:${table}:${String(query.id)}`)
              removes.push({ table, query })
              return { removed: 1 }
            },
            create: async (_table: string, record: Record<string, unknown>) => record,
            transact: async <TInner>(inner: (database: any) => Promise<TInner>) => inner({}),
          })
        },
      },
    },
    moderationStore: {
      appendEvent: async (event: any) => {
        operationLog.push(`append-event:${String(event.type)}`)
        events.push(event)
      },
      createReview: async (record: any) => {
        operationLog.push(`create-review:${String(record.payload?.reportId ?? record.id ?? 'unknown')}`)
        createdReviews.push(record)
        return { id: 'rv-created', ...record }
      },
      listPendingReviews: async () => [],
      updateReportAIResult: async (id: string, aiStatus: string, aiSeverity: string, aiSummary: string | null) => {
        operationLog.push(`update-report:${id}`)
        updatedReports.push({ id, aiStatus, aiSeverity, aiSummary })
      },
    },
    actions: {
      kickMember: async (_bot, guildId: string, channelId: string, memberId: string, permanent: boolean, reason: string) => {
        if (input.kickError) {
          throw input.kickError
        }
        actionCalls.kick.push({ guildId, channelId, memberId, permanent, reason })
      },
    },
    now: () => input.now,
    databaseState: {
      guardUpdates,
      reviewUpdates,
      removes,
    },
    events,
    createdReviews,
    updatedReports,
    operationLog,
    actionCalls,
    botCalls,
    logErrors,
  }
}

function createGuardRecord(overrides: Record<string, unknown> = {}) {
  return {
    id: 'gm-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2001',
    memberName: 'Alice',
    verificationState: 'bound_unverified',
    joinedAt: new Date('2026-04-21T07:30:00.000Z'),
    deadlineAt: new Date('2026-04-21T08:00:00.000Z'),
    mutedAt: new Date('2026-04-21T07:31:00.000Z'),
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: new Date('2026-04-21T07:30:00.000Z'),
    updatedAt: new Date('2026-04-21T07:31:00.000Z'),
    ...overrides,
  }
}

function createReportRecord() {
  return {
    id: 'rp-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    reporterMemberId: '2003',
    targetMemberId: '2002',
    reason: '恶意刷屏',
    aiStatus: 'completed',
    aiSeverity: 'high',
    aiSummary: '高危广告',
    createdAt: new Date('2026-04-21T07:57:00.000Z'),
    updatedAt: new Date('2026-04-21T07:57:00.000Z'),
  }
}

function createReviewRecord(overrides: Record<string, unknown> = {}) {
  return {
    id: 'rv-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2002',
    actionType: 'kick',
    status: 'pending',
    reason: '恶意刷屏',
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt: new Date('2026-04-21T07:55:00.000Z'),
    updatedAt: new Date('2026-04-21T07:55:00.000Z'),
    ...overrides,
  }
}

function createActor() {
  return {
    memberId: 'admin-42',
    displayName: '审核员甲',
  }
}
