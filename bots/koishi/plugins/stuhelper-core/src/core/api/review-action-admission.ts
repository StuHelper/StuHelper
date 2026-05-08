import { h, type Context } from 'koishi'
import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'

import {
  createAdmissionEvent,
  requireGuardRecord,
  resolveManagedBot,
} from './review-action-support'
import { assertConsoleGuildAccess } from './console-guild-scope'
import {
  getNow,
  toErrorMessage,
  type AdmissionActionInput,
  type WorkItemActionActor,
  type WorkItemActionDeps,
} from './review-action-types'

export async function handleAdmissionAction(
  deps: WorkItemActionDeps,
  input: AdmissionActionInput,
  actor: WorkItemActionActor,
) {
  const record = await requireGuardRecord(deps.ctx, input.itemId)
  assertConsoleGuildAccess(actor.guildScope, record.guildId, 'admission work item')
  const now = getNow(deps)

  if (input.action === 'approve') {
    return approveAdmission({ deps, record, actor, now, note: input.note })
  }
  if (input.action === 'deny') {
    return denyAdmission({ deps, record, actor, now, note: input.note })
  }
  return deferAdmission({ deps, record, actor, now, note: input.note })
}

interface AdmissionActionRuntimeInput {
  readonly deps: WorkItemActionDeps
  readonly record: GuardMemberRecord
  readonly actor: WorkItemActionActor
  readonly now: Date
  readonly note?: string
}

async function approveAdmission(input: AdmissionActionRuntimeInput) {
  const { deps, record, actor, now, note } = input
  const claimAt = new Date(now)
  await claimPendingGuardRecord(deps.ctx, record, claimAt)
  const bot = resolveManagedBot(deps.ctx, record.platform, record.botSelfId)
  try {
    await bot.muteGuildMember(record.guildId, record.memberId, 0)
    await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已通过人工准入，已解除限制。`)
    await finalizeGuardRecordRelease({ ctx: deps.ctx, guardId: record.id, claimedAt: claimAt, releasedAt: now })
    await deps.moderationStore.appendEvent(createAdmissionEvent({
      record,
      type: 'join_released',
      level: 'info',
      summaryPrefix: '控制台已放行',
      actor,
      note,
    }))
  } catch (error) {
    await rollbackGuardClaimSafely({ ctx: deps.ctx, guardId: record.id, claimedAt: claimAt, rolledBackAt: getNow(deps), originalError: error })
    throw error
  }
  return `已放行待准入成员：${record.memberId}`
}

async function denyAdmission(input: AdmissionActionRuntimeInput) {
  const { deps, record, actor, now, note } = input
  const claimAt = new Date(now)
  await claimPendingGuardRecord(deps.ctx, record, claimAt)
  const bot = resolveManagedBot(deps.ctx, record.platform, record.botSelfId)
  try {
    await bot.sendMessage(record.channelId, `${h.at(record.memberId)} 已被人工拒绝准入，机器人将移出群聊。`)
    await bot.kickGuildMember(record.guildId, record.memberId)
    await finalizeGuardRecordKick({ ctx: deps.ctx, guardId: record.id, claimedAt: claimAt, kickedAt: now })
    await deps.moderationStore.appendEvent(createAdmissionEvent({
      record,
      type: 'action_executed',
      level: 'high',
      summaryPrefix: '控制台已拒绝准入并移出',
      actor,
      note,
      action: 'deny-admission',
    }))
  } catch (error) {
    await rollbackGuardClaimSafely({ ctx: deps.ctx, guardId: record.id, claimedAt: claimAt, rolledBackAt: getNow(deps), originalError: error })
    throw error
  }
  return `已拒绝待准入成员：${record.memberId}`
}

async function deferAdmission(input: AdmissionActionRuntimeInput) {
  const { deps, record, actor, now, note } = input
  const deferredDeadline = resolveDeferredDeadline(record, now)
  const result = await deps.ctx.database.set(GUARD_MEMBER_TABLE, {
    id: record.id,
    updatedAt: record.updatedAt,
    releasedAt: null,
    kickedAt: null,
  }, { deadlineAt: deferredDeadline, lastError: null, updatedAt: now })
  if (result.matched !== 1) {
    throw new Error(`guard member is already being processed: ${record.id}`)
  }
  await deps.moderationStore.appendEvent(createAdmissionEvent({
    record,
    type: 'action_executed',
    level: 'medium',
    summaryPrefix: '控制台已延期准入处理',
    actor,
    note,
    action: 'defer-admission',
    deadlineAt: deferredDeadline.toISOString(),
  }))
  return `已延期待准入成员：${record.memberId}`
}

function resolveDeferredDeadline(record: GuardMemberRecord, now: Date) {
  const initialWindowMs = record.deadlineAt.getTime() - record.joinedAt.getTime()
  if (initialWindowMs <= 0) {
    throw new Error(`guard record has invalid deadline window: ${record.id}`)
  }
  return new Date(Math.max(now.getTime(), record.deadlineAt.getTime()) + initialWindowMs)
}

async function claimPendingGuardRecord(ctx: Context, record: GuardMemberRecord, claimedAt: Date) {
  const result = await ctx.database.set(GUARD_MEMBER_TABLE, {
    id: record.id,
    updatedAt: record.updatedAt,
    releasedAt: null,
    kickedAt: null,
  }, { updatedAt: claimedAt })
  if (result.matched !== 1) {
    throw new Error(`guard member is already being processed: ${record.id}`)
  }
}

async function finalizeGuardRecordRelease(input: {
  readonly ctx: Context
  readonly guardId: string
  readonly claimedAt: Date
  readonly releasedAt: Date
}) {
  const result = await input.ctx.database.set(GUARD_MEMBER_TABLE, {
    id: input.guardId,
    updatedAt: input.claimedAt,
    releasedAt: null,
    kickedAt: null,
  }, { releasedAt: input.releasedAt, lastError: null, updatedAt: input.releasedAt })
  if (result.matched !== 1) {
    throw new Error(`guard member release lost claim: ${input.guardId}`)
  }
}

async function finalizeGuardRecordKick(input: {
  readonly ctx: Context
  readonly guardId: string
  readonly claimedAt: Date
  readonly kickedAt: Date
}) {
  const result = await input.ctx.database.set(GUARD_MEMBER_TABLE, {
    id: input.guardId,
    updatedAt: input.claimedAt,
    releasedAt: null,
    kickedAt: null,
  }, { kickedAt: input.kickedAt, lastError: null, updatedAt: input.kickedAt })
  if (result.matched !== 1) {
    throw new Error(`guard member kick lost claim: ${input.guardId}`)
  }
}

async function rollbackGuardRecordClaim(input: {
  readonly ctx: Context
  readonly guardId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
  readonly error: unknown
}) {
  const message = input.error instanceof Error ? input.error.message : String(input.error)
  await input.ctx.database.set(GUARD_MEMBER_TABLE, {
    id: input.guardId,
    updatedAt: input.claimedAt,
    releasedAt: null,
    kickedAt: null,
  }, { lastError: message, updatedAt: input.rolledBackAt })
}

async function rollbackGuardClaimSafely(input: {
  readonly ctx: Context
  readonly guardId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
  readonly originalError: unknown
}) {
  try {
    await rollbackGuardRecordClaim({ ...input, error: input.originalError })
  } catch (rollbackError) {
    input.ctx.logger('stuhelperGroupCenter').error(
      'guard rollback failed for %s after business error: %s | original: %s',
      input.guardId,
      toErrorMessage(rollbackError),
      toErrorMessage(input.originalError),
    )
  }
}
