import type { SubscriptionModule } from './subscription.module'

const MUTE_EXPIRE_CHECK_INTERVAL_MS = 60_000
const SECONDS_TO_MS = 1000

export interface MuteExpireBot {
  sendMessage(channelId: string, content: string): Promise<unknown>
}

interface ExpiredMute {
  guildId: string
  userId: string
}

export function setupMuteExpireCheck(host: SubscriptionModule): void {
  host.setCheckInterval(setInterval(() => {
    const bot = host.ctx.bots.values().next().value as MuteExpireBot | undefined
    if (bot) {
      host.checkMuteExpires(bot).catch((error) => {
        host.ctx.logger('stuhelper-core:subscription').error('检查禁言到期失败: %o', error)
      })
    }
  }, MUTE_EXPIRE_CHECK_INTERVAL_MS))

  host.ctx.on('dispose', () => {
    host.clearCheckInterval()
  })
}

export async function checkMuteExpires(
  host: SubscriptionModule,
  bot: MuteExpireBot,
): Promise<void> {
  const expiredMutes = findExpiredMutes(host)
  if (expiredMutes.length === 0) return

  host.data.mutes.flush()
  await notifyMuteExpireSubscribers(host, bot, expiredMutes)
}

function findExpiredMutes(host: SubscriptionModule): ExpiredMute[] {
  const now = Date.now()
  const allMutes = host.data.mutes.getAll()
  const expiredMutes: ExpiredMute[] = []

  for (const [guildId, guildMutes] of Object.entries(allMutes)) {
    for (const [userId, mute] of Object.entries(guildMutes)) {
      const expireAt = mute.startTime + mute.duration * SECONDS_TO_MS
      if (expireAt <= now && !mute.notified) {
        expiredMutes.push({ guildId, userId })
        mute.notified = true
      }
    }
  }

  return expiredMutes
}

async function notifyMuteExpireSubscribers(
  host: SubscriptionModule,
  bot: MuteExpireBot,
  expiredMutes: ExpiredMute[],
): Promise<void> {
  const subscriptions = host.data.subscriptions.getAll().list

  for (const sub of subscriptions) {
    if (!sub.features?.muteExpire) continue
    for (const expired of expiredMutes) {
      if (sub.type === 'group' && sub.id === expired.guildId) {
        try {
          await bot.sendMessage(expired.guildId, `用户 ${expired.userId} 的禁言已到期喵~`)
        } catch (error) {
          host.ctx.logger('stuhelper-core:subscription').error('发送禁言到期通知失败: %o', error)
        }
      }
    }
  }
}
