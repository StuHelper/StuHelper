import { GUARD_MEMBER_TABLE, type GuardMemberRecord } from '@stuhelper/koishi-shared'
import { GuardPolicyStore } from '@stuhelper/koishi-shared'
import {
  MODERATION_REVIEW_TABLE,
  ModerationActionService,
  ModerationStore,
  type ReviewActionType,
  type ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type { Context } from 'koishi'

import { STUHELPER_CONSOLE_SERVICE } from './constants'
import type {
  StuhelperCommandPolicyInput,
  StuhelperGuardBatchActionInput,
  StuhelperGuardBindingInput,
  StuhelperGuardTemplateInput,
  StuhelperMemberRoleInput,
  StuhelperReviewActionInput,
} from './console-types'
import { resolveManagedBot } from './runtime'

export function registerConsoleListeners(ctx: Context) {
  const moderationStore = new ModerationStore(ctx)
  const guardPolicyStore = new GuardPolicyStore(ctx)
  const actions = new ModerationActionService(moderationStore)

  ctx.console.addListener('stuhelper-console/refresh', async () => {
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/guard-action', async (input) => {
    const result = await handleGuardBatchAction(ctx, moderationStore, actions, input)
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
    return result
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/review-action', async (input) => {
    const result = await handleReviewAction(ctx, moderationStore, actions, input)
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
    return result
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/save-keyword-rule', async (input) => {
    const now = new Date()
    const current = (await moderationStore.listAllKeywordRules()).find((item) => item.id === input.id)
    await moderationStore.upsertKeywordRule({
      ...input,
      note: input.note || null,
      enabled: input.enabled ?? true,
      createdAt: current?.createdAt || now,
      updatedAt: now,
    })
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/save-member-roles', async (input) => {
    await saveMemberRoles(moderationStore, input)
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/save-command-policy', async (input) => {
    await saveCommandPolicy(moderationStore, input)
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/save-guard-template', async (input) => {
    const result = await saveGuardTemplate(guardPolicyStore, input)
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
    return result
  }, { authority: 4 })

  ctx.console.addListener('stuhelper-console/save-guard-binding', async (input) => {
    const result = await saveGuardBinding(guardPolicyStore, input)
    await ctx.console.refresh(STUHELPER_CONSOLE_SERVICE)
    return result
  }, { authority: 4 })
}

export async function handleGuardBatchAction(
  ctx: Context,
  moderationStore: ModerationStore,
  actions: ModerationActionService,
  input: StuhelperGuardBatchActionInput,
) {
  const records = (await listActiveGuardMembers(ctx)).filter((item) => input.memberIds.includes(item.id))
  if (!records.length) {
    throw new Error('no guard members selected')
  }

  if (input.action === 'kick') {
    return createGuardReviewRequests(moderationStore, records, input)
  }

  for (const record of records) {
    const bot = resolveManagedBot(ctx, record)
    if (input.action === 'mute') {
      await actions.muteMember(bot, record.guildId, record.channelId, record.memberId, input.seconds || 60, input.reason)
      await updateGuardMember(ctx, record.id, { mutedAt: new Date(), updatedAt: new Date() })
      continue
    }
    if (input.action === 'unmute') {
      await actions.unmuteMember(bot, record.guildId, record.channelId, record.memberId, input.reason)
      continue
    }
    if (input.action === 'set-role') {
      if (!input.roleId) {
        throw new Error('roleId is required for set-role')
      }
      await actions.setMemberRole(bot, record.guildId, record.memberId, input.roleId)
      continue
    }
    if (!input.roleId) {
      throw new Error('roleId is required for unset-role')
    }
    await actions.unsetMemberRole(bot, record.guildId, record.memberId, input.roleId)
  }

  return formatGuardActionResult(input.action, records.length)
}

export async function handleReviewAction(
  ctx: Context,
  moderationStore: ModerationStore,
  actions: ModerationActionService,
  input: StuhelperReviewActionInput,
) {
  const [review] = await ctx.database.get(MODERATION_REVIEW_TABLE, { id: input.reviewId }) as ReviewQueueRecord[]
  if (!review) {
    throw new Error(`review not found: ${input.reviewId}`)
  }
  if (review.status !== 'pending') {
    throw new Error(`review is already resolved: ${input.reviewId}`)
  }

  if (input.action === 'reject') {
    await moderationStore.resolveReview(review.id, 'rejected', 'console', input.note || null)
    await moderationStore.appendEvent({
      platform: review.platform,
      botSelfId: review.botSelfId,
      guildId: review.guildId,
      channelId: review.channelId,
      memberId: review.memberId,
      type: 'review_resolved',
      level: 'info',
      summary: `复核已驳回：${review.memberId}`,
      payload: { reviewId: review.id, note: input.note || null },
    })
    return `已驳回复核：${review.memberId}`
  }

  const bot = resolveManagedBot(ctx, review)
  const permanent = review.actionType === 'kick_and_block'
  await actions.kickMember(bot, review.guildId, review.channelId, review.memberId, permanent, review.reason)
  await moderationStore.resolveReview(review.id, 'executed', 'console', input.note || null)
  await moderationStore.appendEvent({
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    channelId: review.channelId,
    memberId: review.memberId,
    type: 'review_resolved',
    level: 'high',
    summary: `复核已执行：${review.memberId}`,
    payload: { reviewId: review.id, note: input.note || null, actionType: review.actionType },
  })
  await markGuardMemberKicked(ctx, review)
  return `已执行复核动作：${review.memberId}`
}

async function saveMemberRoles(moderationStore: ModerationStore, input: StuhelperMemberRoleInput) {
  await moderationStore.setMemberRoles(input.guildId, input.memberId, input.roles)
}

async function saveCommandPolicy(moderationStore: ModerationStore, input: StuhelperCommandPolicyInput) {
  const now = new Date()
  const current = await moderationStore.getCommandPolicy(input.commandId)
  await moderationStore.upsertCommandPolicy({
    commandId: input.commandId,
    roles: input.roles,
    minAuthority: input.minAuthority,
    createdAt: current?.createdAt || now,
    updatedAt: now,
  })
}

async function saveGuardTemplate(guardPolicyStore: GuardPolicyStore, input: StuhelperGuardTemplateInput) {
  if (!input.id.trim() || !input.name.trim() || !input.reminderTemplate.trim()) {
    throw new Error('guard template id、名称和提醒文案不能为空')
  }
  await guardPolicyStore.saveTemplate({
    ...input,
    id: input.id.trim(),
    name: input.name.trim(),
    reminderTemplate: input.reminderTemplate.trim(),
    exemptUsers: [...input.exemptUsers],
  })
  return `已保存群模板：${input.name}`
}

async function saveGuardBinding(guardPolicyStore: GuardPolicyStore, input: StuhelperGuardBindingInput) {
  if (!input.platform.trim() || !input.guildId.trim() || !input.templateId.trim()) {
    throw new Error('platform、guildId 和 templateId 不能为空')
  }
  const templates = await guardPolicyStore.listTemplates()
  const templateId = input.templateId.trim()
  const template = templates.find((item) => item.id === templateId)
  if (!template) {
    throw new Error(`guard template not found: ${templateId}`)
  }
  await guardPolicyStore.saveBinding({
    ...input,
    platform: input.platform.trim(),
    guildId: input.guildId.trim(),
    templateId,
    note: input.note?.trim() || null,
  })
  return `已保存群绑定：${input.platform.trim()}/${input.guildId.trim()}`
}

async function markGuardMemberKicked(ctx: Context, review: ReviewQueueRecord) {
  const [record] = await ctx.database.get(GUARD_MEMBER_TABLE, {
    platform: review.platform,
    botSelfId: review.botSelfId,
    guildId: review.guildId,
    memberId: review.memberId,
  }) as GuardMemberRecord[]
  if (!record) {
    return
  }
  await updateGuardMember(ctx, record.id, { kickedAt: new Date(), lastError: null, updatedAt: new Date() })
}

async function listActiveGuardMembers(ctx: Context) {
  const records = await ctx.database.get(GUARD_MEMBER_TABLE, {}) as GuardMemberRecord[]
  return records.filter((record) => !record.releasedAt && !record.kickedAt)
}

async function updateGuardMember(ctx: Context, id: string, value: Partial<GuardMemberRecord>) {
  await ctx.database.set(GUARD_MEMBER_TABLE, { id }, value)
}

async function createGuardReviewRequests(
  moderationStore: ModerationStore,
  records: GuardMemberRecord[],
  input: StuhelperGuardBatchActionInput,
) {
  const actionType: ReviewActionType = input.permanent ? 'kick_and_block' : 'kick'

  for (const record of records) {
    await moderationStore.createReview({
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId: record.guildId,
      channelId: record.channelId,
      memberId: record.memberId,
      actionType,
      status: 'pending',
      reason: input.reason,
      operatorMemberId: 'console',
      resolutionNote: null,
      payload: {
        source: 'console-batch-action',
        guardMemberId: record.id,
        permanent: Boolean(input.permanent),
      },
    })
    await moderationStore.appendEvent({
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId: record.guildId,
      channelId: record.channelId,
      memberId: record.memberId,
      type: 'review_created',
      level: 'high',
      summary: `控制台提交了复核申请：${record.memberId}`,
      payload: { actionType, reason: input.reason, permanent: Boolean(input.permanent) },
    })
  }

  return `已提交 ${records.length} 条人工复核申请。`
}

function formatGuardActionResult(action: StuhelperGuardBatchActionInput['action'], count: number) {
  if (action === 'mute') {
    return `已批量禁言 ${count} 名成员。`
  }
  if (action === 'unmute') {
    return `已批量解除 ${count} 名成员的禁言。`
  }
  if (action === 'set-role') {
    return `已批量设置 ${count} 名成员的角色。`
  }
  return `已批量移除 ${count} 名成员的角色。`
}
