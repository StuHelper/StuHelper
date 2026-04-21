import { h, type Context } from 'koishi'

import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'
import {
  MODERATION_REPORT_TABLE,
  type ModerationReportRecord,
  type ReviewActionType,
  type ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'

import {
  createAdmissionEvent,
  createReportActionEvent,
  createReviewResolvedEvent,
  requireGuardRecord,
  requireReportRecord,
  requireReviewRecord,
  resolveManagedBot,
} from './review-action-support'

export type ReviewActionInput = {
  kind: 'review'
  itemId: string
  action: 'execute' | 'reject'
  note?: string
}

export type AdmissionActionInput = {
  kind: 'admission'
  itemId: string
  action: 'approve' | 'deny' | 'defer'
  note?: string
}

export type ReportActionInput = {
  kind: 'report'
  itemId: string
  action: 'dismiss' | 'escalate' | 'create-review'
  note?: string
}

export type WorkItemActionInput =
  | ReviewActionInput
  | AdmissionActionInput
  | ReportActionInput

export interface WorkItemActionDeps {
  ctx: Context
  moderationStore: {
    appendEvent: (input: Record<string, unknown>) => Promise<unknown>
    createReview: (input: Record<string, unknown>) => Promise<{ id: string; actionType: ReviewActionType }>
    resolveReview: (id: string, status: string, operatorMemberId: string, resolutionNote: string | null) => Promise<unknown>
    updateReportAIResult: (id: string, aiStatus: string, aiSeverity: string, aiSummary: string | null) => Promise<unknown>
  }
  actions: {
    kickMember: (
      bot: never,
      guildId: string,
      channelId: string,
      memberId: string,
      permanent: boolean,
      reason: string,
    ) => Promise<unknown>
  }
  now?: () => Date
}

export async function handleWorkItemAction(
  deps: WorkItemActionDeps,
  input: WorkItemActionInput,
) {
  switch (input.kind) {
    case 'review':
      return handleReviewAction(deps, input)
    case 'admission':
      return handleAdmissionAction(deps, input)
    case 'report':
      return handleReportAction(deps, input)
  }
}

async function handleReviewAction(
  deps: WorkItemActionDeps,
  input: ReviewActionInput,
) {
  const review = await requireReviewRecord(deps.ctx, input.itemId)
  if (review.status !== 'pending') {
    throw new Error(`review is already resolved: ${input.itemId}`)
  }

  if (input.action === 'reject') {
    await deps.moderationStore.resolveReview(review.id, 'rejected', 'console', input.note || null)
    await deps.moderationStore.appendEvent(createReviewResolvedEvent(review, 'info', input.note))
    return `已驳回复核：${review.memberId}`
  }

  const bot = resolveManagedBot(deps.ctx, review.platform, review.botSelfId)
  const permanent = review.actionType === 'kick_and_block'
  await deps.actions.kickMember(bot as never, review.guildId, review.channelId, review.memberId, permanent, review.reason)
  await deps.moderationStore.resolveReview(review.id, 'executed', 'console', input.note || null)
  await deps.moderationStore.appendEvent(createReviewResolvedEvent(review, 'high', input.note))
  await markGuardMemberKicked(deps.ctx, review, getNow(deps))
  return `已执行复核动作：${review.memberId}`
}

async function handleAdmissionAction(
  deps: WorkItemActionDeps,
  input: AdmissionActionInput,
) {
  const record = await requireGuardRecord(deps.ctx, input.itemId)
  const now = getNow(deps)

  if (input.action === 'approve') {
    return approveAdmission(deps, record, now, input.note)
  }
  if (input.action === 'deny') {
    return denyAdmission(deps, record, now, input.note)
  }
  return deferAdmission(deps, record, now, input.note)
}

async function approveAdmission(
  deps: WorkItemActionDeps,
  record: GuardMemberRecord,
  now: Date,
  note?: string,
) {
  const bot = resolveManagedBot(deps.ctx, record.platform, record.botSelfId)
  await bot.muteGuildMember(record.guildId, record.memberId, 0)
  await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已通过人工准入，已解除限制。`)
  await deps.ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, { releasedAt: now, lastError: null, updatedAt: now })
  await deps.moderationStore.appendEvent(createAdmissionEvent(record, 'join_released', 'info', '控制台已放行', note))
  return `已放行待准入成员：${record.memberId}`
}

async function denyAdmission(
  deps: WorkItemActionDeps,
  record: GuardMemberRecord,
  now: Date,
  note?: string,
) {
  const bot = resolveManagedBot(deps.ctx, record.platform, record.botSelfId)
  await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已被人工拒绝准入，机器人将移出群聊。`)
  await bot.kickGuildMember(record.guildId, record.memberId)
  await deps.ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, { kickedAt: now, lastError: null, updatedAt: now })
  await deps.moderationStore.appendEvent(createAdmissionEvent(record, 'action_executed', 'high', '控制台已拒绝准入并移出', note, 'deny-admission'))
  return `已拒绝待准入成员：${record.memberId}`
}

async function deferAdmission(
  deps: WorkItemActionDeps,
  record: GuardMemberRecord,
  now: Date,
  note?: string,
) {
  const deferredDeadline = resolveDeferredDeadline(record, now)
  await deps.ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, { deadlineAt: deferredDeadline, lastError: null, updatedAt: now })
  await deps.moderationStore.appendEvent(createAdmissionEvent(record, 'action_executed', 'medium', '控制台已延期准入处理', note, 'defer-admission', deferredDeadline.toISOString()))
  return `已延期待准入成员：${record.memberId}`
}

async function handleReportAction(
  deps: WorkItemActionDeps,
  input: ReportActionInput,
) {
  const report = await requireReportRecord(deps.ctx, input.itemId)

  if (input.action === 'dismiss') {
    await deps.ctx.database.remove(MODERATION_REPORT_TABLE, { id: report.id })
    await deps.moderationStore.appendEvent(createReportActionEvent(report, '控制台已驳回举报', 'info', 'dismiss-report', input.note))
    return `已驳回举报：${report.targetMemberId}`
  }

  if (input.action === 'escalate') {
    const nextSummary = input.note?.trim() || report.aiSummary || report.reason
    await deps.moderationStore.updateReportAIResult(report.id, 'completed', 'high', nextSummary)
    await deps.moderationStore.appendEvent(createReportActionEvent(report, '控制台已升级举报', 'high', 'escalate-report', input.note))
    return `已升级举报：${report.targetMemberId}`
  }

  const review = await deps.moderationStore.createReview({
    platform: report.platform,
    botSelfId: report.botSelfId,
    guildId: report.guildId,
    channelId: report.channelId,
    memberId: report.targetMemberId,
    actionType: selectReportReviewAction(report),
    status: 'pending',
    reason: report.reason,
    operatorMemberId: 'console',
    resolutionNote: null,
    payload: {
      source: 'console-report',
      reportId: report.id,
      aiSeverity: report.aiSeverity,
      aiSummary: report.aiSummary,
      note: input.note || null,
    },
  })
  await deps.ctx.database.remove(MODERATION_REPORT_TABLE, { id: report.id })
  await deps.moderationStore.appendEvent({
    platform: report.platform,
    botSelfId: report.botSelfId,
    guildId: report.guildId,
    channelId: report.channelId,
    memberId: report.targetMemberId,
    type: 'review_created',
    level: 'high',
    summary: `控制台已把举报转入复核：${report.targetMemberId}`,
    payload: { actionType: review.actionType, reportId: report.id, reviewId: review.id, source: 'console' },
  })
  return `已把举报转成复核：${report.targetMemberId}`
}

function resolveDeferredDeadline(record: GuardMemberRecord, now: Date) {
  const initialWindowMs = record.deadlineAt.getTime() - record.joinedAt.getTime()
  if (initialWindowMs <= 0) {
    throw new Error(`guard record has invalid deadline window: ${record.id}`)
  }
  return new Date(Math.max(now.getTime(), record.deadlineAt.getTime()) + initialWindowMs)
}

function selectReportReviewAction(report: ModerationReportRecord): ReviewActionType {
  return report.aiSeverity === 'high' ? 'kick_and_block' : 'kick'
}

async function markGuardMemberKicked(ctx: Context, review: ReviewQueueRecord, now: Date) {
  const [record] = await ctx.database.get(GUARD_MEMBER_TABLE, {
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    memberId: review.memberId,
  }) as GuardMemberRecord[]
  if (!record) {
    return
  }
  await ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, { kickedAt: now, lastError: null, updatedAt: now })
}

function getNow(deps: WorkItemActionDeps) {
  return deps.now ? deps.now() : new Date()
}
