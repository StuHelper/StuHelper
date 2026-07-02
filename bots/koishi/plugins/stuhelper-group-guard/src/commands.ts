import type { Context, Session } from 'koishi'

import {
  COMMAND_POLICY_IDS,
  canExecuteCommand,
  createFallbackCommandPolicy,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import type {
  AdmissionRuntimeSettingsStore,
  GroupGuardBehaviorSettingsStore,
  GroupGuardFunSettings,
} from '@stuhelper/koishi-shared'
import {
  DEFAULT_GROUP_GUARD_FUN_SETTINGS,
  renderMessageTemplate,
  resolveGroupGuardMessages,
} from '@stuhelper/koishi-shared'

import type { ReportService } from './report-service'
import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
  type GroupGuardMessages,
} from './group-guard-message-provider'

interface CommandDeps {
  store: ModerationStore
  reportService: ReportService
  runtimeSettings?: AdmissionRuntimeSettingsStore
  behaviorSettings?: GroupGuardBehaviorSettingsStore
  messageProvider?: GroupGuardMessageProvider
}

export function registerPublicCommands(ctx: Context, deps: CommandDeps) {
  const messages = resolveGroupGuardMessages()
  ctx.command('举报 <targetMemberId> <reason:text>', renderMessageTemplate(messages.publicReportCommandDescription))
    .action(async ({ session }, targetMemberId, reason) => {
      if (!session) {
        throw new Error('report command requires session')
      }
      const disabled = await ensurePublicCommandsEnabled(deps)
      if (disabled) return disabled
      const denial = await ensureCommandAccess(deps, session, COMMAND_POLICY_IDS.report)
      if (denial) {
        return denial
      }
      if (!targetMemberId?.trim() || !reason?.trim()) {
        return groupGuardMessage(
          await getGroupGuardMessages(deps.messageProvider),
          'publicReportMissingArgs',
        )
      }
      return deps.reportService.handleReport(session, targetMemberId.trim(), reason.trim())
    })

  ctx.command('骰子 [sides:natural]', renderMessageTemplate(messages.diceCommandDescription))
    .action(async ({ session }, sides) => {
      const disabled = await ensurePublicCommandsEnabled(deps)
      if (disabled) return disabled
      const denial = await ensureCommandAccess(deps, session, COMMAND_POLICY_IDS.dice)
      if (denial) {
        return denial
      }
      const fun = await getFunSettings(deps)
      const resolvedSides = clampSides(sides || fun.diceSides)
      const result = randomInt(1, resolvedSides)
      return buildDiceMessage(
        session,
        resolvedSides,
        result,
        await getGroupGuardMessages(deps.messageProvider),
      )
    })

  ctx.command('抽禁言', renderMessageTemplate(messages.muteLotteryCommandDescription))
    .action(async ({ session }) => {
      if (!session) {
        throw new Error('mute lottery command requires session')
      }
      const disabled = await ensurePublicCommandsEnabled(deps)
      if (disabled) return disabled
      const denial = await ensureCommandAccess(
        deps,
        session,
        COMMAND_POLICY_IDS.muteLottery,
      )
      if (denial) {
        return denial
      }
      return handleMuteLottery(session, deps)
    })
}

async function ensurePublicCommandsEnabled(deps: CommandDeps) {
  if (deps.runtimeSettings && !await deps.runtimeSettings.isPublicCommandsEnabled()) {
    return groupGuardMessage(
      await getGroupGuardMessages(deps.messageProvider),
      'publicCommandsDisabled',
    )
  }
}

async function handleMuteLottery(session: Session, deps: CommandDeps) {
  const messages = await getGroupGuardMessages(deps.messageProvider)
  const guildId = session.guildId
  const channelId = session.channelId
  if (!guildId || !channelId) {
    return groupGuardMessage(messages, 'muteLotteryGroupOnly')
  }

  const profile = await deps.store.getFunProfile(session.userId)
  const fun = await getFunSettings(deps)
  const now = new Date()
  const drawCount = (profile?.muteDrawCount || 0) + 1
  const guaranteed = drawCount >= fun.muteLotteryPityThreshold
  const rolled = guaranteed
    ? fun.muteLotteryPitySeconds
    : randomInt(1, fun.muteLotteryBaseSeconds)
  const seconds = Math.min(rolled, fun.muteLotteryMaxSeconds)

  await deps.store.saveFunProfile({
    memberId: session.userId,
    muteDrawCount: guaranteed ? 0 : drawCount,
    pityBonus: guaranteed ? 0 : drawCount,
    lastDrawAt: now,
    createdAt: profile?.createdAt || now,
    updatedAt: now,
  })
  await session.bot.muteGuildMember(guildId, session.userId, seconds * 1000)
  await deps.store.appendEvent({
    platform: session.platform,
    botSelfId: session.selfId,
    guildId,
    channelId,
    memberId: session.userId,
    type: 'action_executed',
    level: guaranteed ? 'high' : 'low',
    summary: groupGuardMessage(messages, 'moderationMuteLotteryEventSummary', {
      memberId: session.userId,
    }),
    payload: { seconds, guaranteed },
  })
  return renderMessageTemplate(guaranteed ? messages.muteLotteryPityResult : messages.muteLotteryResult, {
    memberId: session.userId,
    seconds,
  })
}

function clampSides(sides: number) {
  return Math.max(2, Math.min(1000, sides))
}

function buildDiceMessage(
  session: Session | undefined,
  sides: number,
  result: number,
  messages: GroupGuardMessages,
) {
  const memberId = session?.userId || 'unknown'
  return groupGuardMessage(messages, 'diceResult', { memberId, sides, result })
}

function randomInt(min: number, max: number) {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

async function getFunSettings(deps: CommandDeps): Promise<GroupGuardFunSettings> {
  return deps.behaviorSettings
    ? deps.behaviorSettings.getFunSettings()
    : DEFAULT_GROUP_GUARD_FUN_SETTINGS
}

async function ensureCommandAccess(deps: CommandDeps, session: Session | undefined, commandId: string) {
  const guildId = session?.guildId
  if (!session || !guildId) {
    return
  }
  const [policy, memberRoles] = await Promise.all([
    deps.store.getCommandPolicy(commandId),
    deps.store.getMemberRoles(guildId, session.userId),
  ])
  const allowed = canExecuteCommand({
    authority: resolveAuthority(session),
    memberRoles,
    // 公开命令：无策略记录时显式声明对所有成员开放。
    policy: policy ?? createFallbackCommandPolicy(commandId, 0),
  })
  if (allowed) {
    return
  }
  return groupGuardMessage(
    await getGroupGuardMessages(deps.messageProvider),
    'commandAccessDenied',
  )
}

function resolveAuthority(session: Session | undefined) {
  const target = session as { user?: { authority?: number } } | undefined
  return target?.user?.authority ?? 0
}
