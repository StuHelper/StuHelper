import type { Context, Session } from 'koishi'

import {
  COMMAND_POLICY_IDS,
  canExecuteCommand,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import type { StuhelperGroupGuardPluginConfig } from '@stuhelper/koishi-shared'

import type { ReportService } from './report-service'

interface CommandDeps {
  store: ModerationStore
  reportService: ReportService
  config: StuhelperGroupGuardPluginConfig
}

export function registerPublicCommands(ctx: Context, deps: CommandDeps) {
  ctx.command('举报 <targetMemberId> <reason:text>', '举报群成员并触发审核')
    .action(async ({ session }, targetMemberId, reason) => {
      if (!session) {
        throw new Error('report command requires session')
      }
      const denial = await ensureCommandAccess(deps.store, session, COMMAND_POLICY_IDS.report)
      if (denial) {
        return denial
      }
      if (!targetMemberId?.trim() || !reason?.trim()) {
        return '请提供被举报成员 ID 和举报原因。'
      }
      return deps.reportService.handleReport(session, targetMemberId.trim(), reason.trim())
    })

  ctx.command('骰子 [sides:natural]', '投掷骰子')
    .action(async ({ session }, sides) => {
      const denial = await ensureCommandAccess(deps.store, session, COMMAND_POLICY_IDS.dice)
      if (denial) {
        return denial
      }
      const resolvedSides = clampSides(sides || deps.config.fun.diceSides)
      const result = randomInt(1, resolvedSides)
      return buildDiceMessage(session, resolvedSides, result)
    })

  ctx.command('抽禁言', '随机抽取自己的禁言时长，带保底机制')
    .action(async ({ session }) => {
      if (!session) {
        throw new Error('mute lottery command requires session')
      }
      const denial = await ensureCommandAccess(
        deps.store,
        session,
        COMMAND_POLICY_IDS.muteLottery,
      )
      if (denial) {
        return denial
      }
      return handleMuteLottery(session, deps)
    })
}

async function handleMuteLottery(session: Session, deps: CommandDeps) {
  const guildId = session.guildId
  const channelId = session.channelId
  if (!guildId || !channelId) {
    return '抽禁言只能在群聊中使用。'
  }

  const profile = await deps.store.getFunProfile(session.userId)
  const now = new Date()
  const drawCount = (profile?.muteDrawCount || 0) + 1
  const guaranteed = drawCount >= deps.config.fun.muteLotteryPityThreshold
  const rolled = guaranteed
    ? deps.config.fun.muteLotteryPitySeconds
    : randomInt(1, deps.config.fun.muteLotteryBaseSeconds)
  const seconds = Math.min(rolled, deps.config.fun.muteLotteryMaxSeconds)

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
    summary: `${session.userId} 触发抽禁言`,
    payload: { seconds, guaranteed },
  })
  return guaranteed
    ? `保底触发，${session.userId} 本次自助禁言 ${seconds} 秒。`
    : `${session.userId} 本次自助禁言 ${seconds} 秒。`
}

function clampSides(sides: number) {
  return Math.max(2, Math.min(1000, sides))
}

function buildDiceMessage(session: Session | undefined, sides: number, result: number) {
  const memberId = session?.userId || 'unknown'
  return `${memberId} 投出了 d${sides} = ${result}`
}

function randomInt(min: number, max: number) {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

async function ensureCommandAccess(store: ModerationStore, session: Session | undefined, commandId: string) {
  const guildId = session?.guildId
  if (!session || !guildId) {
    return
  }
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    store.getMemberRoles(guildId, session.userId),
  ])
  const allowed = canExecuteCommand({
    authority: resolveAuthority(session),
    memberRoles,
    policy,
  })
  if (allowed) {
    return
  }
  return '命令权限不足。'
}

function resolveAuthority(session: Session | undefined) {
  const target = session as { user?: { authority?: number } } | undefined
  return target?.user?.authority ?? 0
}
