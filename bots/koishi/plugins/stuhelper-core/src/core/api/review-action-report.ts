import {
  MODERATION_REPORT_TABLE,
  type ModerationReportRecord,
  type ReviewActionType,
} from '@stuhelper/koishi-moderation-core'

import {
  createReportActionEvent,
  requireReportRecord,
} from './review-action-support'
import { assertConsoleGuildAccess } from './console-guild-scope'
import {
  REVIEW_STATUS_PENDING,
  type ReportActionInput,
  type WorkItemActionActor,
  type WorkItemActionDeps,
} from './review-action-types'

export async function handleReportAction(
  deps: WorkItemActionDeps,
  input: ReportActionInput,
  actor: WorkItemActionActor,
) {
  const report = await requireReportRecord(deps.ctx, input.itemId)
  assertConsoleGuildAccess(actor.guildScope, report.guildId, 'report work item')
  const store = resolveTransactionalStore(deps)

  if (input.action === 'dismiss') {
    return dismissReport({ store, report, actor, note: input.note })
  }
  if (input.action === 'escalate') {
    return escalateReport({ store, report, actor, note: input.note })
  }
  return createReviewFromReport({ store, report, actor, note: input.note })
}

interface ReportActionRuntimeInput {
  readonly store: TransactionalModerationStore
  readonly report: ModerationReportRecord
  readonly actor: WorkItemActionActor
  readonly note?: string
}

async function dismissReport(input: ReportActionRuntimeInput) {
  const { store, report, actor, note } = input
  await store.transact(async (tx) => {
    await tx.removeReport(report.id)
    await tx.appendEvent(createReportActionEvent({
      report,
      summaryPrefix: '控制台已驳回举报',
      level: 'info',
      action: 'dismiss-report',
      actor,
      note,
    }))
  })
  return `已驳回举报：${report.targetMemberId}`
}

async function escalateReport(input: ReportActionRuntimeInput) {
  const { store, report, actor, note } = input
  const nextSummary = note?.trim() || report.aiSummary || report.reason
  await store.transact(async (tx) => {
    await tx.updateReportAIResult({
      id: report.id,
      aiStatus: 'completed',
      aiSeverity: 'high',
      aiSummary: nextSummary,
    })
    await tx.appendEvent(createReportActionEvent({
      report,
      summaryPrefix: '控制台已升级举报',
      level: 'high',
      action: 'escalate-report',
      actor,
      note,
    }))
  })
  return `已升级举报：${report.targetMemberId}`
}

async function createReviewFromReport(input: ReportActionRuntimeInput) {
  const { store, report, actor, note } = input
  await store.transact(async (tx) => {
    const review = await tx.createReview(buildReportReviewInput(report, actor, note))
    await tx.removeReport(report.id)
    await tx.appendEvent(buildReportReviewCreatedEvent({ report, actor, review, note }))
  })
  return `已把举报转成复核：${report.targetMemberId}`
}

interface TransactionalModerationStore {
  readonly appendEvent: (input: Record<string, unknown>) => Promise<unknown>
  readonly createReview: (input: Record<string, unknown>) => Promise<{ id: string; actionType: ReviewActionType }>
  readonly removeReport: (id: string) => Promise<unknown>
  readonly transact: <T>(callback: (store: TransactionalModerationStore) => Promise<T>) => Promise<T>
  readonly updateReportAIResult: (input: {
    readonly id: string
    readonly aiStatus: string
    readonly aiSeverity: string
    readonly aiSummary: string | null
  }) => Promise<unknown>
}

function resolveTransactionalStore(deps: WorkItemActionDeps): TransactionalModerationStore {
  if (deps.moderationStore.transact && deps.moderationStore.removeReport) {
    return deps.moderationStore as TransactionalModerationStore
  }

  return {
    ...deps.moderationStore,
    removeReport: async (id: string) => deps.ctx.database.remove(MODERATION_REPORT_TABLE, { id }),
    transact: async <T>(callback: (store: TransactionalModerationStore) => Promise<T>) => {
      return deps.ctx.database.transact(async () => callback({
        ...deps.moderationStore,
        removeReport: async (id: string) => deps.ctx.database.remove(MODERATION_REPORT_TABLE, { id }),
      } as TransactionalModerationStore))
    },
  } as TransactionalModerationStore
}

function buildReportReviewInput(
  report: ModerationReportRecord,
  actor: WorkItemActionActor,
  note?: string,
) {
  return {
    platform: report.platform,
    botSelfId: report.botSelfId,
    guildId: report.guildId,
    channelId: report.channelId,
    memberId: report.targetMemberId,
    actionType: selectReportReviewAction(report),
    status: REVIEW_STATUS_PENDING,
    reason: report.reason,
    operatorMemberId: actor.memberId,
    resolutionNote: null,
    payload: {
      source: 'console-report',
      reportId: report.id,
      aiSeverity: report.aiSeverity,
      aiSummary: report.aiSummary,
      note: note || null,
      operatorMemberId: actor.memberId,
      operatorName: actor.displayName,
    },
  }
}

function buildReportReviewCreatedEvent(input: {
  readonly report: ModerationReportRecord
  readonly actor: WorkItemActionActor
  readonly review: { id: string; actionType: ReviewActionType }
  readonly note?: string
}) {
  const { report, actor, review, note } = input
  return {
    platform: report.platform,
    botSelfId: report.botSelfId,
    guildId: report.guildId,
    channelId: report.channelId,
    memberId: report.targetMemberId,
    type: 'review_created',
    level: 'high',
    summary: `控制台已把举报转入复核：${report.targetMemberId}`,
    payload: {
      actionType: review.actionType,
      reportId: report.id,
      reviewId: review.id,
      source: 'console',
      note: note || null,
      operatorMemberId: actor.memberId,
      operatorName: actor.displayName,
    },
  }
}

function selectReportReviewAction(report: ModerationReportRecord): ReviewActionType {
  return report.aiSeverity === 'high' ? 'kick_and_block' : 'kick'
}
