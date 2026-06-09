import type { Context, Session } from 'koishi'

import {
  COMMAND_POLICY_IDS,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
  type AdmissionRuntimeSettingsStore,
  type AdminRuntimeSettingsStore,
  PlatformAPIError,
  renderMessageTemplate,
  resolveAdminMessages,
  type FreshmanApplication,
  type PlatformClient,
} from '@stuhelper/koishi-shared'

import { ensureAdminCommandAccess } from './command-access'

const EXTENSION_PATTERN = /^\+([1-9]\d*)d$/i

interface FreshmanReviewCommandDeps {
  readonly moderationStore: ModerationStore
  readonly platform: PlatformClient
  readonly runtimeSettings?: AdmissionRuntimeSettingsStore
  readonly adminSettings?: AdminRuntimeSettingsStore
}

type AdminMessages = ReturnType<typeof resolveAdminMessages>

export function registerFreshmanReviewCommands(ctx: Context, deps: FreshmanReviewCommandDeps) {
  const messages = resolveAdminMessages()

  ctx.command('新生审核查看 <applicationID:text>', renderMessageTemplate(messages.freshmanViewCommandDescription), { authority: 3 })
    .action(({ session }, applicationID) => handleView(deps, session, applicationID))
  ctx.command('新生审核通过 <payload:text>', renderMessageTemplate(messages.freshmanApproveCommandDescription), { authority: 3 })
    .action(({ session }, payload) => handleApprove(deps, session, payload))
  ctx.command('新生审核驳回 <payload:text>', renderMessageTemplate(messages.freshmanRejectCommandDescription), { authority: 3 })
    .action(({ session }, payload) => handleReject(deps, session, payload))
  ctx.command('新生黑名单解除 <payload:text>', renderMessageTemplate(messages.freshmanBlacklistReleaseCommandDescription), { authority: 3 })
    .action(({ session }, payload) => handleBlacklistRelease(deps, session, payload))
}

async function handleView(deps: FreshmanReviewCommandDeps, session: Session | undefined, applicationID: string) {
  const messages = await getAdminMessages(deps)
  const input = await commandInput(deps, session, messages)
  if ('error' in input) return input.error
  const id = applicationID?.trim()
  if (!id) return adminMessage(messages, 'freshmanMissingApplicationID')
  return runAdmissionCommand(messages, async () => {
    const application = await deps.platform.viewFreshmanApplication(id, input.context)
    return formatFreshmanApplication(application, messages)
  })
}

async function handleApprove(deps: FreshmanReviewCommandDeps, session: Session | undefined, payload: string) {
  const messages = await getAdminMessages(deps)
  const parsed = parseApprovePayload(payload, messages)
  if ('error' in parsed) return parsed.error
  const input = await commandInput(deps, session, messages)
  if ('error' in input) return input.error
  return runAdmissionCommand(messages, async () => {
    await deps.platform.reviewFreshmanApplication(parsed.applicationID, {
      ...input.context,
      action: 'approve',
      expiresInDays: parsed.expiresInDays,
    })
    return parsed.expiresInDays
      ? adminMessage(messages, 'freshmanApproveSuccessWithExtension', {
          applicationID: parsed.applicationID,
          expiresInDays: parsed.expiresInDays,
        })
      : adminMessage(messages, 'freshmanApproveSuccess', { applicationID: parsed.applicationID })
  })
}

async function handleReject(deps: FreshmanReviewCommandDeps, session: Session | undefined, payload: string) {
  const messages = await getAdminMessages(deps)
  const parsed = parseRejectPayload(payload, messages)
  if ('error' in parsed) return parsed.error
  const input = await commandInput(deps, session, messages)
  if ('error' in input) return input.error
  return runAdmissionCommand(messages, async () => {
    await deps.platform.reviewFreshmanApplication(parsed.applicationID, {
      ...input.context,
      action: 'reject',
      reason: parsed.reason,
    })
    return adminMessage(messages, 'freshmanRejectSuccess', { applicationID: parsed.applicationID })
  })
}

async function handleBlacklistRelease(deps: FreshmanReviewCommandDeps, session: Session | undefined, payload: string) {
  const messages = await getAdminMessages(deps)
  const input = await commandInput(deps, session, messages)
  if ('error' in input) return input.error
  const parsed = parseBlacklistReleasePayload(payload, messages)
  if ('error' in parsed) return parsed.error
  return runAdmissionCommand(messages, async () => {
    await deps.platform.releaseMemberBlacklistBySubject({
      platform: input.platform,
      subjectType: 'qq_user',
      subjectID: parsed.qqID,
      scopeType: parsed.scopeType,
      guildID: parsed.guildID,
      releaseReasonCode: 'manual_pardon',
      releaseReason: 'freshman review command release',
      operatorQQID: input.context.operatorQQID,
    })
    return adminMessage(messages, 'freshmanBlacklistReleaseSuccess', {
      qqID: parsed.qqID,
      scope: formatBlacklistScope(parsed, messages),
    })
  })
}

async function commandInput(
  deps: FreshmanReviewCommandDeps,
  session: Session | undefined,
  messages: AdminMessages,
) {
  if (!session?.guildId) return { error: adminMessage(messages, 'freshmanManagementGroupOnly') } as const
  const denial = await ensureAdminCommandAccess({
    store: deps.moderationStore,
    session,
    commandId: COMMAND_POLICY_IDS.guardReviews,
    targetGuildId: session.guildId,
    runtimeSettings: deps.runtimeSettings,
    messages,
  })
  if (denial) return { error: denial } as const
  return {
    platform: session.platform,
    context: {
      operatorQQID: session.userId,
      guildID: session.guildId,
      channelID: session.channelId,
      rawCommand: session.content?.trim() || '',
    },
  } as const
}

async function runAdmissionCommand(
  messages: AdminMessages,
  action: () => Promise<string>,
) {
  try {
    return await action()
  } catch (error) {
    return formatAdmissionCommandError(error, messages)
  }
}

function parseApprovePayload(
  payload: string | undefined,
  messages: AdminMessages,
) {
  const parts = splitPayload(payload)
  if (!parts.length) return { error: adminMessage(messages, 'freshmanMissingApplicationID') } as const
  if (parts.length === 1) return { applicationID: parts[0] } as const
  if (parts.length > 2) return { error: adminMessage(messages, 'freshmanApproveInvalidFormat') } as const
  const match = parts[1].match(EXTENSION_PATTERN)
  if (!match) return { error: adminMessage(messages, 'freshmanApproveInvalidExtension') } as const
  return { applicationID: parts[0], expiresInDays: Number(match[1]) } as const
}

function parseRejectPayload(
  payload: string | undefined,
  messages: AdminMessages,
) {
  const parts = splitPayload(payload)
  if (parts.length < 2) return { error: adminMessage(messages, 'freshmanRejectInvalidFormat') } as const
  return { applicationID: parts[0], reason: parts.slice(1).join(' ') } as const
}

function splitPayload(payload: string | undefined) {
  return (payload || '').trim().split(/\s+/).filter(Boolean)
}

function parseBlacklistReleasePayload(
  payload: string | undefined,
  messages: AdminMessages,
) {
  const parts = splitPayload(payload)
  if (parts.length < 2) return { error: adminMessage(messages, 'freshmanBlacklistReleaseInvalidFormat') } as const
  if (parts.length > 2) return { error: adminMessage(messages, 'freshmanBlacklistReleaseInvalidFormat') } as const
  if (parts[1].toLowerCase() === 'global') {
    return { qqID: parts[0], scopeType: 'global' as const }
  }
  return { qqID: parts[0], scopeType: 'guild' as const, guildID: parts[1] }
}

function formatBlacklistScope(
  input: { scopeType: 'global' | 'guild'; guildID?: string },
  messages: AdminMessages,
) {
  return input.scopeType === 'global'
    ? adminMessage(messages, 'freshmanBlacklistScopeGlobal')
    : adminMessage(messages, 'freshmanBlacklistScopeGuild', { guildID: input.guildID })
}

function formatFreshmanApplication(
  application: FreshmanApplication,
  messages: AdminMessages,
) {
  return adminMessage(messages, 'freshmanApplicationSummary', {
    applicationID: application.id,
    status: application.status,
    applicantName: application.applicantNameMasked,
    schoolID: application.schoolID,
    departmentOrMajor: application.departmentOrMajor || adminMessage(messages, 'freshmanApplicationDepartmentFallback'),
    provisionalExpiresAt: application.provisionalExpiresAt || adminMessage(messages, 'freshmanApplicationExpiryFallback'),
  })
}

function formatAdmissionCommandError(
  error: unknown,
  messages: AdminMessages,
) {
  if (error instanceof PlatformAPIError && error.status === 403) {
    return admissionForbiddenReply(error, messages)
  }
  return adminMessage(messages, 'freshmanCommandFailed', {
    error: error instanceof Error ? error.message : String(error),
  })
}

function admissionForbiddenReply(
  error: PlatformAPIError,
  messages: AdminMessages,
) {
  if (error.code === 'admission.operator_qq_unbound') {
    return adminMessage(messages, 'freshmanOperatorQQUnbound')
  }
  if (error.code === 'admission.operator_forbidden') {
    return adminMessage(messages, 'freshmanOperatorForbidden')
  }
  if (error.code === 'admission.management_guild_forbidden') {
    return adminMessage(messages, 'freshmanManagementGuildForbidden')
  }
  return adminMessage(messages, 'freshmanBackendForbidden', { message: error.message })
}

function adminMessage(
  messages: AdminMessages,
  key: keyof ReturnType<typeof resolveAdminMessages>,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(messages[key], variables)
}

async function getAdminMessages(deps: FreshmanReviewCommandDeps): Promise<AdminMessages> {
  return deps.adminSettings?.getMessages() ?? resolveAdminMessages()
}
