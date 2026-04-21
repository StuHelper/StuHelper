import assert from 'node:assert/strict'
import test from 'node:test'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'
import {
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
} from '@stuhelper/koishi-moderation-core'

import { handleWorkItemAction } from './review-actions'

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
  })

  assert.equal(message, `已放行待准入成员：${guardRecord.memberId}`)
  assert.deepEqual(deps.botCalls.mute, [[guardRecord.guildId, guardRecord.memberId, 0]])
  assert.equal(deps.databaseState.guardUpdates[0].query.id, guardRecord.id)
  assert.ok(deps.databaseState.guardUpdates[0].patch.releasedAt instanceof Date)
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
  })

  assert.equal(message, `已把举报转成复核：${report.targetMemberId}`)
  assert.equal(deps.createdReviews.length, 1)
  assert.equal(deps.createdReviews[0].actionType, 'kick_and_block')
  assert.deepEqual(deps.databaseState.removes, [{ table: MODERATION_REPORT_TABLE, query: { id: report.id } }])
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
  })

  assert.equal(message, `已驳回复核：${review.memberId}`)
  assert.deepEqual(deps.resolvedReviews, [{
    id: review.id,
    status: 'rejected',
    operatorMemberId: 'console',
    resolutionNote: '证据不足',
  }])
  assert.equal(deps.botCalls.kick.length, 0)
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
  })

  assert.equal(message, `已延期待准入成员：${guardRecord.memberId}`)
  assert.equal(
    String(deps.databaseState.guardUpdates[0].patch.deadlineAt),
    new Date('2026-04-21T08:30:00.000Z').toString(),
  )
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
  })

  assert.equal(message, `已驳回举报：${report.targetMemberId}`)
  assert.deepEqual(deps.databaseState.removes, [{ table: MODERATION_REPORT_TABLE, query: { id: report.id } }])
})

function createActionDeps(input: {
  guardRecords: any[]
  reports: any[]
  reviews?: any[]
  now: Date
}) {
  const guardUpdates: Array<{ query: Record<string, unknown>; patch: Record<string, unknown> }> = []
  const removes: Array<{ table: string; query: Record<string, unknown> }> = []
  const events: any[] = []
  const createdReviews: any[] = []
  const resolvedReviews: Array<{ id: string; status: string; operatorMemberId: string; resolutionNote: string | null }> = []
  const botCalls = {
    mute: [] as Array<[string, string, number]>,
    kick: [] as Array<[string, string]>,
    send: [] as Array<[string, string]>,
  }
  const bots = [{
    platform: 'onebot',
    selfId: 'bot',
    muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
      botCalls.mute.push([guildId, memberId, duration])
    },
    kickGuildMember: async (guildId: string, memberId: string) => {
      botCalls.kick.push([guildId, memberId])
    },
    sendMessage: async (channelId: string, content: string) => {
      botCalls.send.push([channelId, content])
    },
  }]

  return {
    ctx: {
      bots,
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
          if (table === GUARD_MEMBER_TABLE) {
            guardUpdates.push({ query, patch })
          }
        },
        remove: async (table: string, query: Record<string, unknown>) => {
          removes.push({ table, query })
        },
      },
    },
    moderationStore: {
      appendEvent: async (event: any) => {
        events.push(event)
      },
      createReview: async (record: any) => {
        createdReviews.push(record)
        return { id: 'rv-created', ...record }
      },
      listPendingReviews: async () => [],
      resolveReview: async (id: string, status: string, operatorMemberId: string, resolutionNote: string | null) => {
        resolvedReviews.push({ id, status, operatorMemberId, resolutionNote })
      },
      updateReportAIResult: async () => undefined,
    },
    actions: {
      kickMember: async () => undefined,
    },
    now: () => input.now,
    databaseState: {
      guardUpdates,
      removes,
    },
    events,
    createdReviews,
    resolvedReviews,
    botCalls,
  }
}

function createGuardRecord() {
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

function createReviewRecord() {
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
  }
}
