import { h, type Universal } from 'koishi'

import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

import type { ModerationStore } from './store'
import type { ModerationRuntimeRef } from './types'

export type ModerationBot = Universal.Methods & {
  platform?: string
  selfId: string
}

export type ModerationMessageProvider = () =>
  Partial<StuhelperGroupGuardMessageConfig> |
  Promise<Partial<StuhelperGroupGuardMessageConfig>>

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
  constructor(
    private readonly store: ModerationStore,
    private readonly messages?: Partial<StuhelperGroupGuardMessageConfig> | ModerationMessageProvider,
  ) {}

  async warnMember(input: WarnMemberInput) {
    const messages = await this.getMessages()
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
      summary: moderationMessage(messages, 'moderationWarnEventSummary', {
        memberId: input.memberId,
        reason: input.reason,
      }),
      payload: { totalWarnings: warning.total, reason: input.reason },
    })
    return warning
  }

  async muteMember(input: MuteMemberInput) {
    const messages = await this.getMessages()
    await input.bot.muteGuildMember(input.guildId, input.memberId, input.seconds * 1000)
    const message = renderMessageTemplate(messages.moderationMuteNotice, {
      at: h.at(input.memberId),
      memberId: input.memberId,
      reason: input.reason,
      seconds: input.seconds,
    })
    if (message) {
      await input.bot.sendMessage(input.channelId, message)
    }
    await this.store.appendEvent({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'high',
      summary: moderationMessage(messages, 'moderationMuteEventSummary', {
        memberId: input.memberId,
        reason: input.reason,
        seconds: input.seconds,
      }),
      payload: { seconds: input.seconds, reason: input.reason },
    })
  }

  async unmuteMember(input: BotMemberActionInput) {
    const messages = await this.getMessages()
    await input.bot.muteGuildMember(input.guildId, input.memberId, 0)
    const message = renderMessageTemplate(messages.moderationUnmuteNotice, {
      at: h.at(input.memberId),
      memberId: input.memberId,
      reason: input.reason,
    })
    if (message) {
      await input.bot.sendMessage(input.channelId, message)
    }
    await this.store.appendEvent({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'medium',
      summary: moderationMessage(messages, 'moderationUnmuteEventSummary', {
        memberId: input.memberId,
        reason: input.reason,
      }),
      payload: { reason: input.reason },
    })
  }

  async kickMember(input: KickMemberInput) {
    const messages = await this.getMessages()
    const message = renderMessageTemplate(messages.moderationKickNotice, {
      at: h.at(input.memberId),
      memberId: input.memberId,
      reason: input.reason,
    })
    if (message) {
      await input.bot.sendMessage(input.channelId, message)
    }
    await input.bot.kickGuildMember(input.guildId, input.memberId, input.permanent)
    await this.store.appendEvent({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.memberId,
      type: 'action_executed',
      level: 'critical',
      summary: moderationMessage(messages, 'moderationKickEventSummary', {
        memberId: input.memberId,
        reason: input.reason,
        permanent: input.permanent,
      }),
      payload: { permanent: input.permanent, reason: input.reason },
    })
  }

  async setMemberRole(input: MemberRoleInput) {
    await input.bot.setGuildMemberRole(input.guildId, input.memberId, input.roleId)
  }

  async unsetMemberRole(input: MemberRoleInput) {
    await input.bot.unsetGuildMemberRole(input.guildId, input.memberId, input.roleId)
  }

  private async getMessages() {
    const messages = typeof this.messages === 'function'
      ? await this.messages()
      : this.messages
    return resolveGroupGuardMessages(messages)
  }
}

function moderationMessage(
  messages: ReturnType<typeof resolveGroupGuardMessages>,
  key: keyof ReturnType<typeof resolveGroupGuardMessages>,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(messages[key], variables)
}
