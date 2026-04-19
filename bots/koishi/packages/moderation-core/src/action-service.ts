import { h, type Universal } from 'koishi'

import type { ModerationStore } from './store'
import type { ModerationRuntimeRef } from './types'

export type ModerationBot = Universal.Methods & {
  platform?: string
  selfId: string
}

export class ModerationActionService {
  constructor(private readonly store: ModerationStore) {}

  async warnMember(runtime: ModerationRuntimeRef, guildId: string, channelId: string, memberId: string, reason: string) {
    const now = new Date()
    const warning = await this.store.incrementWarning(guildId, memberId, reason, now)
    await this.store.appendEvent({
      platform: runtime.platform,
      botSelfId: runtime.botSelfId,
      guildId,
      channelId,
      memberId,
      type: 'action_executed',
      level: 'medium',
      summary: `已对 ${memberId} 记警告：${reason}`,
      payload: { totalWarnings: warning.total, reason },
    })
    return warning
  }

  async muteMember(bot: ModerationBot, guildId: string, channelId: string, memberId: string, seconds: number, reason: string) {
    await bot.muteGuildMember(guildId, memberId, seconds * 1000)
    await bot.sendMessage(channelId, `${h.at(memberId)} 因 ${reason} 被禁言 ${seconds} 秒。`)
    await this.store.appendEvent({
      platform: bot.platform || '',
      botSelfId: bot.selfId,
      guildId,
      channelId,
      memberId,
      type: 'action_executed',
      level: 'high',
      summary: `已禁言 ${memberId}`,
      payload: { seconds, reason },
    })
  }

  async unmuteMember(bot: ModerationBot, guildId: string, channelId: string, memberId: string, reason: string) {
    await bot.muteGuildMember(guildId, memberId, 0)
    await bot.sendMessage(channelId, `${h.at(memberId)} 已解除禁言。原因：${reason}`)
    await this.store.appendEvent({
      platform: bot.platform || '',
      botSelfId: bot.selfId,
      guildId,
      channelId,
      memberId,
      type: 'action_executed',
      level: 'medium',
      summary: `已解除 ${memberId} 的禁言`,
      payload: { reason },
    })
  }

  async kickMember(bot: ModerationBot, guildId: string, channelId: string, memberId: string, permanent: boolean, reason: string) {
    await bot.sendMessage(channelId, `${h.at(memberId)} 因 ${reason} 将被移出群聊。`)
    await bot.kickGuildMember(guildId, memberId, permanent)
    await this.store.appendEvent({
      platform: bot.platform || '',
      botSelfId: bot.selfId,
      guildId,
      channelId,
      memberId,
      type: 'action_executed',
      level: 'critical',
      summary: `已移出 ${memberId}`,
      payload: { permanent, reason },
    })
  }

  async setMemberRole(bot: ModerationBot, guildId: string, memberId: string, roleId: string) {
    await bot.setGuildMemberRole(guildId, memberId, roleId)
  }

  async unsetMemberRole(bot: ModerationBot, guildId: string, memberId: string, roleId: string) {
    await bot.unsetGuildMemberRole(guildId, memberId, roleId)
  }
}
