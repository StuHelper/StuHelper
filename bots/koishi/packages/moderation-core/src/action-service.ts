import { h, type Universal } from 'koishi'

import type { ModerationStore } from './store'
import type { ModerationRuntimeRef } from './types'

export type ModerationBot = Universal.Methods & {
  platform?: string
  selfId: string
}

export interface WarnMemberInput {
  readonly runtime: ModerationRuntimeRef
  readonly guildId: string
  readonly channelId: string
  readonly memberId: string
  readonly reason: string
}

export interface BotMemberActionInput {
  readonly bot: ModerationBot
  readonly guildId: string
  readonly channelId: string
  readonly memberId: string
  readonly reason: string
}

export interface MuteMemberInput extends BotMemberActionInput {
  readonly seconds: number
}

export interface KickMemberInput extends BotMemberActionInput {
  readonly permanent: boolean
}

export interface MemberRoleInput {
  readonly bot: ModerationBot
  readonly guildId: string
  readonly memberId: string
  readonly roleId: string
}

export class ModerationActionService {
  constructor(private readonly store: ModerationStore) {}

  async warnMember(input: WarnMemberInput) {
    const now = new Date()
    const warning = await this.store.incrementWarning({
      guildId: input.guildId,
      memberId: input.memberId,
      reason: input.reason,
      now,
    })
    await this.store.appendEvent({
      platform: input.runtime.platform,
      botSelfId: input.runtime.botSelfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'medium',
      summary: `已对 ${input.memberId} 记警告：${input.reason}`,
      payload: { totalWarnings: warning.total, reason: input.reason },
    })
    return warning
  }

  async muteMember(input: MuteMemberInput) {
    await input.bot.muteGuildMember(input.guildId, input.memberId, input.seconds * 1000)
    await input.bot.sendMessage(input.channelId, `${h.at(input.memberId)} 因 ${input.reason} 被禁言 ${input.seconds} 秒。`)
    await this.store.appendEvent({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'high',
      summary: `已禁言 ${input.memberId}`,
      payload: { seconds: input.seconds, reason: input.reason },
    })
  }

  async unmuteMember(input: BotMemberActionInput) {
    await input.bot.muteGuildMember(input.guildId, input.memberId, 0)
    await input.bot.sendMessage(input.channelId, `${h.at(input.memberId)} 已解除禁言。原因：${input.reason}`)
    await this.store.appendEvent({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'medium',
      summary: `已解除 ${input.memberId} 的禁言`,
      payload: { reason: input.reason },
    })
  }

  async kickMember(input: KickMemberInput) {
    await input.bot.sendMessage(input.channelId, `${h.at(input.memberId)} 因 ${input.reason} 将被移出群聊。`)
    await input.bot.kickGuildMember(input.guildId, input.memberId, input.permanent)
    await this.store.appendEvent({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'critical',
      summary: `已移出 ${input.memberId}`,
      payload: { permanent: input.permanent, reason: input.reason },
    })
  }

  async setMemberRole(input: MemberRoleInput) {
    await input.bot.setGuildMemberRole(input.guildId, input.memberId, input.roleId)
  }

  async unsetMemberRole(input: MemberRoleInput) {
    await input.bot.unsetGuildMemberRole(input.guildId, input.memberId, input.roleId)
  }
}
