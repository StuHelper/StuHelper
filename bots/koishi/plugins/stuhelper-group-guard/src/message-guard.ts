import { h, type Logger, type Session } from 'koishi'

import {
  ModerationActionService,
  ModerationStore,
  createMessageLedgerPreview,
  detectRepeatTrigger,
  matchKeywordRules,
  normalizeModerationContent,
} from '@stuhelper/koishi-moderation-core'
import type {
  GroupGuardBehaviorSettingsStore,
  GroupGuardModerationSettings,
} from '@stuhelper/koishi-shared'
import {
  DEFAULT_GROUP_GUARD_MODERATION_SETTINGS,
  evaluateThresholdExpression,
} from '@stuhelper/koishi-shared'

import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
  type GroupGuardMessages,
} from './group-guard-message-provider'

interface MessageGuardDeps {
  store: ModerationStore
  actions: ModerationActionService
  logger: Logger
  behaviorSettings?: GroupGuardBehaviorSettingsStore
  messageProvider?: GroupGuardMessageProvider
}

type KeywordRuleHit = ReturnType<typeof matchKeywordRules>[number]

interface MessageModerationInput {
  readonly session: Session
  readonly guildId: string
  readonly channelId: string
  readonly messageId: string
  readonly content: string
  readonly normalizedContent: string
}

interface KeywordHitInput extends MessageModerationInput {
  readonly hit: KeywordRuleHit
}

export class MessageGuardService {
  constructor(private readonly deps: MessageGuardDeps) {}

  async handleMessage(session: Session) {
    const guildId = normalizeID(session.guildId || session.event.guild?.id)
    const channelId = normalizeID(session.channelId || session.event.channel?.id)
    const content = (session.content || session.event.message?.content || '').trim()
    if (!guildId || !channelId || !content) {
      return
    }

    const messageId = resolveMessageID(session)
    const normalizedContent = normalizeModerationContent(content)
    await this.deps.store.saveMessage({
      messageId,
      platform: session.platform,
      botSelfId: session.selfId,
      guildId,
      channelId,
      memberId: session.userId,
      content,
      normalizedContent,
      quoteMessageId: session.quote?.id || null,
      createdAt: new Date(),
      deletedAt: null,
    })

    await this.processKeywordHits({ session, guildId, channelId, messageId, content, normalizedContent })
    await this.processRepeatHit({ session, guildId, channelId, messageId, content, normalizedContent })
  }

  async handleMessageDeleted(session: Session) {
    const guildId = normalizeID(session.guildId || session.event.guild?.id)
    const channelId = normalizeID(session.channelId || session.event.channel?.id)
    const messageId = resolveMessageID(session)
    if (!guildId || !channelId || !messageId) {
      return
    }

    const record = await this.deps.store.getMessage(messageId)
    if (!record) {
      return
    }

    const messages = await this.getMessages()
    await this.deps.store.markMessageDeleted(messageId, new Date())
    await this.deps.store.appendEvent({
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId,
      channelId,
      memberId: record.memberId,
      type: 'message_deleted',
      level: 'medium',
      summary: groupGuardMessage(messages, 'moderationAntiRecallEventSummary', {
        memberId: record.memberId,
      }),
      payload: { content: record.content, preview: createMessageLedgerPreview(record) },
    })

    const moderation = await this.getModerationSettings()
    if (moderation.antiRecallNotify) {
      const message = groupGuardMessage(messages, 'antiRecallNotify', {
        at: h.at(record.memberId),
        memberId: record.memberId,
        content: record.content,
      })
      if (message) {
        await session.bot.sendMessage(channelId, message)
      }
    }
  }

  private async processKeywordHits(input: MessageModerationInput) {
    const rules = await this.listEffectiveKeywordRules(input.guildId)
    const hits = matchKeywordRules(rules, {
      guildId: input.guildId,
      content: input.content,
      normalizedContent: input.normalizedContent,
    }, ({ rule, error }) => {
      this.deps.logger.warn('keyword rule skipped due to evaluation failure', {
        ruleId: rule.id,
        guildId: input.guildId,
        error: error instanceof Error ? error.message : String(error),
      })
    })
    if (!hits.length) {
      return
    }

    for (const hit of hits) {
      await this.processKeywordHit({ ...input, hit })
    }
  }

  private async processKeywordHit(input: KeywordHitInput) {
    const messages = await this.getMessages()
    await this.appendKeywordHitEvent(input, messages)
    if (input.hit.action === 'delete') {
      await input.session.bot.deleteMessage(input.channelId, input.messageId)
    }
    const moderation = await this.getModerationSettings()
    const warning = await this.warnKeywordHit(input, messages)
    if (this.shouldMuteKeywordHit(input.hit, warning.total, moderation)) {
      await this.muteKeywordHit(input, moderation, messages)
    }
    if (input.hit.action === 'review') {
      await this.createKeywordReview(input, messages)
    }
  }

  private async appendKeywordHitEvent(input: KeywordHitInput, messages: GroupGuardMessages) {
    await this.deps.store.appendEvent({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      type: 'keyword_hit',
      level: 'high',
      summary: groupGuardMessage(messages, 'moderationKeywordHitEventSummary', {
        memberId: input.session.userId,
        ruleId: input.hit.ruleId,
      }),
      payload: { action: input.hit.action, note: input.hit.note },
    })
  }

  private async warnKeywordHit(input: KeywordHitInput, messages: GroupGuardMessages) {
    return this.deps.actions.warnMember({
      runtime: { platform: input.session.platform, botSelfId: input.session.selfId },
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      reason: input.hit.note || groupGuardMessage(messages, 'moderationKeywordHitReason'),
    })
  }

  private shouldMuteKeywordHit(
    hit: KeywordRuleHit,
    warnings: number,
    moderation: GroupGuardModerationSettings,
  ) {
    const escalated = evaluateThresholdExpression(moderation.warningThresholdExpression, {
      warnings,
      repeats: 0,
      reports: 0,
    })
    return hit.action === 'mute' || escalated
  }

  private async muteKeywordHit(
    input: KeywordHitInput,
    moderation: GroupGuardModerationSettings,
    messages: GroupGuardMessages,
  ) {
    const muteSeconds = input.hit.action === 'mute'
      ? input.hit.muteSeconds || moderation.defaultMuteSeconds
      : moderation.defaultMuteSeconds
    await this.deps.actions.muteMember({
      bot: input.session.bot,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      seconds: muteSeconds,
      reason: groupGuardMessage(messages, 'moderationKeywordRuleHitReason'),
    })
  }

  private async createKeywordReview(input: KeywordHitInput, messages: GroupGuardMessages) {
    await this.deps.store.createReview({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      actionType: 'kick',
      status: 'pending',
      reason: input.hit.note || groupGuardMessage(messages, 'moderationKeywordRuleHitReason'),
      operatorMemberId: null,
      resolutionNote: null,
      payload: { ruleId: input.hit.ruleId },
    })
  }

  private async processRepeatHit(input: MessageModerationInput) {
    const messages = await this.getMessages()
    const moderation = await this.getModerationSettings()
    const records = await this.deps.store.listRecentMessages(input.guildId, moderation.repeatWindowSize)
    const previous = records
      .filter((record) => record.messageId !== input.messageId)
      .reverse()
      .map((record) => ({ normalizedContent: record.normalizedContent, memberId: record.memberId }))
    const repeat = detectRepeatTrigger(previous, input.normalizedContent, moderation.repeatThreshold)
    if (!repeat.hit) {
      return
    }

    const warning = await this.deps.actions.warnMember({
      runtime: { platform: input.session.platform, botSelfId: input.session.selfId },
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      reason: groupGuardMessage(messages, 'moderationRepeatHitReason'),
    })
    await this.deps.store.appendEvent({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      type: 'repeat_hit',
      level: 'high',
      summary: groupGuardMessage(messages, 'moderationRepeatHitEventSummary', {
        memberId: input.session.userId,
        count: repeat.count,
      }),
      payload: { count: repeat.count },
    })
    const shouldMute = evaluateThresholdExpression(moderation.warningThresholdExpression, {
      warnings: warning.total,
      repeats: repeat.count,
      reports: 0,
    })
    if (shouldMute) {
      await this.deps.actions.muteMember({
        bot: input.session.bot,
        guildId: input.guildId,
        channelId: input.channelId,
        memberId: input.session.userId,
        seconds: moderation.defaultMuteSeconds,
        reason: groupGuardMessage(messages, 'moderationRepeatAutoMuteReason'),
      })
    }
  }

  private async listEffectiveKeywordRules(guildId: string) {
    return this.deps.store.listKeywordRules(guildId)
  }

  private async getModerationSettings() {
    return this.deps.behaviorSettings
      ? this.deps.behaviorSettings.getModerationSettings()
      : DEFAULT_GROUP_GUARD_MODERATION_SETTINGS
  }

  private async getMessages() {
    return getGroupGuardMessages(this.deps.messageProvider)
  }
}

function resolveMessageID(session: Session) {
  return normalizeID(session.messageId || session.event.message?.id || session.id)
}

function normalizeID(value: string | number | undefined) {
  if (value === undefined) {
    return ''
  }
  return String(value)
}
