import type { Context, Session } from 'koishi'

import {
  COMMAND_POLICY_IDS,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
  PlatformAPIError,
  type FreshmanApplication,
  type FreshmanCommandContext,
  type PlatformClient,
} from '@stuhelper/koishi-shared'

import { ensureAdminCommandAccess } from './command-access'

const EXTENSION_PATTERN = /^\+([1-9]\d*)d$/i

interface FreshmanReviewCommandDeps {
  readonly moderationStore: ModerationStore
  readonly platform: PlatformClient
}

export function registerFreshmanReviewCommands(ctx: Context, deps: FreshmanReviewCommandDeps) {
  ctx.command('新生审核查看 <applicationID:text>', '查看新生认证申请', { authority: 3 })
    .action(({ session }, applicationID) => handleView(deps, session, applicationID))
  ctx.command('新生审核通过 <payload:text>', '通过新生认证申请', { authority: 3 })
    .action(({ session }, payload) => handleApprove(deps, session, payload))
  ctx.command('新生审核驳回 <payload:text>', '驳回新生认证申请', { authority: 3 })
    .action(({ session }, payload) => handleReject(deps, session, payload))
  ctx.command('新生黑名单解除 <payload:text>', '解除新生入群认证黑名单', { authority: 3 })
    .action(({ session }, payload) => handleBlacklistRelease(deps, session, payload))
}

async function handleView(deps: FreshmanReviewCommandDeps, session: Session | undefined, applicationID: string) {
  const input = await commandInput(deps, session)
  if ('error' in input) return input.error
  const id = applicationID?.trim()
  if (!id) return '请提供申请 ID。'
  return runAdmissionCommand(async () => {
    const application = await deps.platform.viewFreshmanApplication(id, input.context)
    return formatFreshmanApplication(application)
  })
}

async function handleApprove(deps: FreshmanReviewCommandDeps, session: Session | undefined, payload: string) {
  const parsed = parseApprovePayload(payload)
  if ('error' in parsed) return parsed.error
  const input = await commandInput(deps, session)
  if ('error' in input) return input.error
  return runAdmissionCommand(async () => {
    await deps.platform.reviewFreshmanApplication(parsed.applicationID, {
      ...input.context,
      action: 'approve',
      expiresInDays: parsed.expiresInDays,
    })
    return parsed.expiresInDays
      ? `已通过新生认证申请 ${parsed.applicationID}，临时身份 ${parsed.expiresInDays} 天后过期。`
      : `已通过新生认证申请 ${parsed.applicationID}。`
  })
}

async function handleReject(deps: FreshmanReviewCommandDeps, session: Session | undefined, payload: string) {
  const parsed = parseRejectPayload(payload)
  if ('error' in parsed) return parsed.error
  const input = await commandInput(deps, session)
  if ('error' in input) return input.error
  return runAdmissionCommand(async () => {
    await deps.platform.reviewFreshmanApplication(parsed.applicationID, {
      ...input.context,
      action: 'reject',
      reason: parsed.reason,
    })
    return `已驳回新生认证申请 ${parsed.applicationID}。`
  })
}

async function handleBlacklistRelease(deps: FreshmanReviewCommandDeps, session: Session | undefined, payload: string) {
  const input = await commandInput(deps, session)
  if ('error' in input) return input.error
  const parsed = parseBlacklistReleasePayload(payload)
  if ('error' in parsed) return parsed.error
  return runAdmissionCommand(async () => {
    await deps.platform.releaseMemberBlacklistBySubject({
      platform: session!.platform,
      subjectType: 'qq_user',
      subjectID: parsed.qqID,
      scopeType: parsed.scopeType,
      guildID: parsed.guildID,
      releaseReasonCode: 'manual_pardon',
      releaseReason: 'freshman review command release',
      operatorQQID: input.context.operatorQQID,
    })
    return `已解除 ${parsed.qqID} 的${formatBlacklistScope(parsed)}入群认证黑名单。`
  })
}

async function commandInput(deps: FreshmanReviewCommandDeps, session: Session | undefined) {
  if (!session?.guildId) return { error: '请在新生审核管理群中执行此命令。' } as const
  const denial = await ensureAdminCommandAccess({
    store: deps.moderationStore,
    session,
    commandId: COMMAND_POLICY_IDS.guardReviews,
    targetGuildId: session.guildId,
  })
  if (denial) return { error: denial } as const
  return {
    context: {
      operatorQQID: session.userId,
      guildID: session.guildId,
      channelID: session.channelId,
      rawCommand: session.content?.trim() || '',
    },
  } as const
}

async function runAdmissionCommand(action: () => Promise<string>) {
  try {
    return await action()
  } catch (error) {
    return formatAdmissionCommandError(error)
  }
}

function parseApprovePayload(payload: string | undefined) {
  const parts = splitPayload(payload)
  if (!parts.length) return { error: '请提供申请 ID。' } as const
  if (parts.length === 1) return { applicationID: parts[0] } as const
  if (parts.length > 2) return { error: '通过命令格式为：新生审核通过 <申请ID> [+30d]。' } as const
  const match = parts[1].match(EXTENSION_PATTERN)
  if (!match) return { error: '审批延长天数格式应为 +30d，且必须是正整数天数。' } as const
  return { applicationID: parts[0], expiresInDays: Number(match[1]) } as const
}

function parseRejectPayload(payload: string | undefined) {
  const parts = splitPayload(payload)
  if (parts.length < 2) return { error: '驳回命令格式为：新生审核驳回 <申请ID> <原因>。' } as const
  return { applicationID: parts[0], reason: parts.slice(1).join(' ') } as const
}

function splitPayload(payload: string | undefined) {
  return (payload || '').trim().split(/\s+/).filter(Boolean)
}

function parseBlacklistReleasePayload(payload: string | undefined) {
  const parts = splitPayload(payload)
  if (parts.length < 2) return { error: '解除命令格式为：新生黑名单解除 <QQ号> <群号|global>。' } as const
  if (parts.length > 2) return { error: '解除命令格式为：新生黑名单解除 <QQ号> <群号|global>。' } as const
  if (parts[1].toLowerCase() === 'global') {
    return { qqID: parts[0], scopeType: 'global' as const }
  }
  return { qqID: parts[0], scopeType: 'guild' as const, guildID: parts[1] }
}

function formatBlacklistScope(input: { scopeType: 'global' | 'guild'; guildID?: string }) {
  return input.scopeType === 'global' ? '全局' : `群 ${input.guildID} 的`
}

function formatFreshmanApplication(application: FreshmanApplication) {
  return [
    `申请 ${application.id}`,
    `状态：${application.status}`,
    `姓名：${application.applicantNameMasked}`,
    `学校ID：${application.schoolID}`,
    `专业：${application.departmentOrMajor || '未提供'}`,
    `临时身份过期：${application.provisionalExpiresAt || '未设置'}`,
  ].join('\n')
}

function formatAdmissionCommandError(error: unknown) {
  if (error instanceof PlatformAPIError && error.status === 403) {
    return admissionForbiddenReply(error)
  }
  return error instanceof Error
    ? `新生审核命令执行失败：${error.message}`
    : `新生审核命令执行失败：${String(error)}`
}

function admissionForbiddenReply(error: PlatformAPIError) {
  if (error.code === 'admission.operator_qq_unbound') {
    return '你的 QQ 未绑定 StuHelper 管理员账号，请先完成管理员 QQ 绑定。'
  }
  if (error.code === 'admission.operator_forbidden') {
    return '你的 StuHelper 账号没有新生审核权限。'
  }
  if (error.code === 'admission.management_guild_forbidden') {
    return '当前群不在新生审核管理群白名单内。'
  }
  return `后端拒绝执行该审核命令：${error.message}`
}
