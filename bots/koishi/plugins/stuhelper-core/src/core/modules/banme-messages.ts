import type { Session } from 'koishi'

import type { BanMeRecord } from '../../types'
import type { JackpotResult } from './banme-types'

export function formatSuccessLog(timeStr: string, jackpot: JackpotResult, record: BanMeRecord): string {
  return `成功：${timeStr} (Jackpot: ${jackpot.isJackpot}, Pity: ${record.pity}, Count: ${record.count})`
}

export function formatBanmeMessage(input: {
  readonly session: Session
  readonly isAuto: boolean
  readonly timeStr: string
  readonly jackpot: JackpotResult
  readonly record: BanMeRecord
}): string {
  const { session, isAuto, timeStr, jackpot, record } = input
  let message = isAuto
    ? `🎲 检测到使用特殊字符逃避禁言，抽到了 ${timeStr} 的禁言喵！\n`
    : `🎲 ${session.username} 抽到了 ${timeStr} 的禁言喵！\n`

  if (jackpot.isJackpot) {
    message += record.guaranteed
      ? '【金】呜呜呜歪掉了！但是下次一定会中的喵！\n'
      : '【金】喵喵喵！恭喜主人中了UP！\n'
    if (jackpot.isGuaranteed) message += '触发保底啦喵~\n'
  }

  return message
}
