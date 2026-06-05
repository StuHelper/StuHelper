import { Time } from 'koishi'

import type { DataManager } from '../data'
import type { PushMessageBot } from '../services'
import { formatDuration } from '../../utils'
import {
  DEFAULT_LEAVE_CONFIG,
  DEFAULT_LEAVE_COOLDOWN_DAYS,
  MUTE_EXPIRE_CHECK_INTERVAL_MS,
  eventLogger,
  groupConfigOf,
  type EventRuntimeHost,
  type EventSession,
} from './event-support'

export function setupEventScheduledTasks(host: EventRuntimeHost): void {
  host.ctx.setInterval(async () => {
    const bot = firstPushMessageBot(host.ctx.bots.values())
    if (bot) {
      await checkMuteExpires(host, bot)
    }
  }, MUTE_EXPIRE_CHECK_INTERVAL_MS)
}

export async function handleGuildMemberAdded(
  host: EventRuntimeHost,
  session: EventSession,
) {
  await resumeMuteAfterRejoin(host, session)

  const message = `[成员加入] 用户 ${session.userId} 加入了群 ${session.guildId}`
  await host.ctx.stuhelperGroupCenter.pushMessage(session.bot, message, 'memberChange')
}

export async function handleGuildMemberRemoved(
  host: EventRuntimeHost,
  session: EventSession,
) {
  recordLeaveCooldown(host, session)
  recordRemainingMute(host, session)

  const message = `[成员退出] 用户 ${session.userId} 退出了群 ${session.guildId}`
  await host.ctx.stuhelperGroupCenter.pushMessage(session.bot, message, 'memberChange')
}

async function resumeMuteAfterRejoin(host: EventRuntimeHost, session: EventSession) {
  const mutes = host.data.mutes.getAll()
  const muteRecord = mutes[session.guildId]?.[session.userId]
  if (!muteRecord?.leftGroup) return

  try {
    await session.bot.muteGuildMember(session.guildId, session.userId, muteRecord.remainingTime)
    muteRecord.startTime = Date.now()
    muteRecord.duration = muteRecord.remainingTime
    delete muteRecord.remainingTime
    muteRecord.leftGroup = false
    host.data.mutes.set(session.guildId, mutes[session.guildId])

    await session.send(`检测到未完成的禁言，继续执行剩余 ${formatDuration(muteRecord.duration)} 的禁言`)
  } catch (error) {
    eventLogger.error('恢复禁言失败:', error)
  }
}

function recordLeaveCooldown(host: EventRuntimeHost, session: EventSession): void {
  const groupConfig = groupConfigOf(host, session.guildId, DEFAULT_LEAVE_CONFIG)
  const leaveCooldown = groupConfig.leaveCooldown ?? DEFAULT_LEAVE_COOLDOWN_DAYS
  if (leaveCooldown <= DEFAULT_LEAVE_COOLDOWN_DAYS) return

  const leaveRecordKey = `${session.guildId}_${session.userId}`
  const leaveRecords = host.data.leaveRecords.getAll()
  leaveRecords[leaveRecordKey] = {
    expireTime: Date.now() + leaveCooldown * Time.day,
  }
  host.data.leaveRecords.set(leaveRecordKey, leaveRecords[leaveRecordKey])
}

function recordRemainingMute(host: EventRuntimeHost, session: EventSession): void {
  const mutes = host.data.mutes.getAll()
  const muteRecord = mutes[session.guildId]?.[session.userId]
  if (!muteRecord) return

  const elapsedTime = Date.now() - muteRecord.startTime
  if (elapsedTime < muteRecord.duration) {
    muteRecord.remainingTime = muteRecord.duration - elapsedTime
    muteRecord.leftGroup = true
    host.data.mutes.set(session.guildId, mutes[session.guildId])
    return
  }

  delete mutes[session.guildId][session.userId]
  host.data.mutes.set(session.guildId, mutes[session.guildId])
}

export async function checkMuteExpires(host: EventRuntimeHost, bot: PushMessageBot): Promise<void> {
  try {
    const mutes = host.data.mutes.getAll()
    const expiredMutes = findExpiredMutes(mutes, Date.now())

    for (const { guildId, userId } of expiredMutes) {
      const message = `[禁言到期] 群 ${guildId} 用户 ${userId} 的禁言已到期`
      await host.ctx.stuhelperGroupCenter.pushMessage(bot, message, 'muteExpire')
      delete mutes[guildId][userId]
      host.data.mutes.set(guildId, mutes[guildId])
    }
  } catch (error) {
    eventLogger.error('检查禁言到期失败:', error)
  }
}

function firstPushMessageBot(bots: Iterable<unknown>): PushMessageBot | null {
  for (const bot of bots) {
    if (isPushMessageBot(bot)) return bot
  }
  return null
}

function isPushMessageBot(value: unknown): value is PushMessageBot {
  if (typeof value !== 'object' || value === null) return false

  const bot = value as Record<string, unknown>
  return typeof bot.sendMessage === 'function' && typeof bot.sendPrivateMessage === 'function'
}

function findExpiredMutes(
  mutes: ReturnType<DataManager['mutes']['getAll']>,
  now: number,
): { guildId: string; userId: string }[] {
  const expiredMutes: { guildId: string; userId: string }[] = []

  for (const guildId in mutes) {
    for (const userId in mutes[guildId]) {
      const record = mutes[guildId][userId]
      if (!record.leftGroup && record.startTime + record.duration <= now) {
        expiredMutes.push({ guildId, userId })
      }
    }
  }

  return expiredMutes
}
