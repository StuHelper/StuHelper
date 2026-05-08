import { h, type Logger, type Session } from 'koishi'

import {
  ModerationActionService,
  ModerationStore,
  createMessageLedgerPreview,
  detectRepeatTrigger,
  evaluateThresholdExpression,
  matchKeywordRules,
  normalizeModerationContent,
} from '@stuhelper/koishi-moderation-core'
import type { StuhelperGroupGuardPluginConfig, StuhelperKeywordRuleConfig } from '@stuhelper/koishi-shared'

interface MessageGuardDeps {
  store: ModerationStore
  actions: ModerationActionService
  logger: Logger
  config: StuhelperGroupGuardPluginConfig
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

  async bootstrapKeywordRules() {
    const now = new Date()
    for (const rule of this.deps.config.moderation.keywordRules) {
      await this.deps.store.upsertKeywordRule({
        ...rule,
        enabled: rule.enabled ?? true,
        note: rule.note || null,
        createdAt: now,
        updatedAt: now,
      })
    }
  }

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

    await this.deps.store.markMessageDeleted(messageId, new Date())
    await this.deps.store.appendEvent({
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId,
      channelId,
      memberId: record.memberId,
      type: 'message_deleted',
      level: 'medium',
      summary: `检测到 ${record.memberId} 撤回消息`,
      payload: { content: record.content, preview: createMessageLedgerPreview(record) },
    })

    if (this.deps.config.moderation.antiRecallNotify) {
      await session.bot.sendMessage(channelId, `${h.at(record.memberId)} 检测到撤回消息：${record.content}`)
    }
  }

  private async processKeywordHits(input: MessageModerationInput) {
    const rules = await this.listEffectiveKeywordRules(input.guildId)
    const hits = matchKeywordRules(rules, {
      guildId: input.guildId,
      content: input.content,
      normalizedContent: input.normalizedContent,
    })
    if (!hits.length) {
      return
    }

    for (const hit of hits) {
      await this.processKeywordHit({ ...input, hit })
    }
  }

  private async processKeywordHit(input: KeywordHitInput) {
    await this.appendKeywordHitEvent(input)
    if (input.hit.action === 'delete') {
      await input.session.bot.deleteMessage(input.channelId, input.messageId)
    }
    const warning = await this.warnKeywordHit(input)
    if (this.shouldMuteKeywordHit(input.hit, warning.total)) {
      await this.muteKeywordHit(input)
    }
    if (input.hit.action === 'review') {
      await this.createKeywordReview(input)
    }
  }

  private async appendKeywordHitEvent(input: KeywordHitInput) {
    await this.deps.store.appendEvent({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      type: 'keyword_hit',
      level: 'high',
      summary: `${input.session.userId} 命中关键词规则 ${input.hit.ruleId}`,
      payload: { action: input.hit.action, note: input.hit.note },
    })
  }

  private async warnKeywordHit(input: KeywordHitInput) {
    return this.deps.actions.warnMember({
      runtime: { platform: input.session.platform, botSelfId: input.session.selfId },
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      reason: input.hit.note || '关键词命中',
    })
  }

  private shouldMuteKeywordHit(hit: KeywordRuleHit, warnings: number) {
    const escalated = evaluateThresholdExpression(this.deps.config.moderation.warningThresholdExpression, {
      warnings,
      repeats: 0,
      reports: 0,
    })
    return hit.action === 'mute' || escalated
  }

  private async muteKeywordHit(input: KeywordHitInput) {
    const muteSeconds = input.hit.action === 'mute'
      ? input.hit.muteSeconds || this.deps.config.moderation.defaultMuteSeconds
      : this.deps.config.moderation.defaultMuteSeconds
    await this.deps.actions.muteMember({
      bot: input.session.bot,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      seconds: muteSeconds,
      reason: '关键词规则命中',
    })
  }

  private async createKeywordReview(input: KeywordHitInput) {
    await this.deps.store.createReview({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      actionType: 'kick',
      status: 'pending',
      reason: input.hit.note || '关键词规则命中',
      operatorMemberId: null,
      resolutionNote: null,
      payload: { ruleId: input.hit.ruleId },
    })
  }

  private async processRepeatHit(input: MessageModerationInput) {
    const records = await this.deps.store.listRecentMessages(input.guildId, this.deps.config.moderation.repeatWindowSize)
    const previous = records
      .filter((record) => record.messageId !== input.messageId)
      .reverse()
      .map((record) => ({ normalizedContent: record.normalizedContent, memberId: record.memberId }))
    const repeat = detectRepeatTrigger(previous, input.normalizedContent, this.deps.config.moderation.repeatThreshold)
    if (!repeat.hit) {
      return
    }

    const warning = await this.deps.actions.warnMember({
      runtime: { platform: input.session.platform, botSelfId: input.session.selfId },
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      reason: '复读命中',
    })
    await this.deps.store.appendEvent({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.session.userId,
      type: 'repeat_hit',
      level: 'high',
      summary: `${input.session.userId} 触发复读检测`,
      payload: { count: repeat.count },
    })
    const shouldMute = evaluateThresholdExpression(this.deps.config.moderation.warningThresholdExpression, {
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
        seconds: this.deps.config.moderation.defaultMuteSeconds,
        reason: '复读规则触发自动处罚',
      })
    }
  }

  private async listEffectiveKeywordRules(guildId: string) {
    const storedRules = await this.deps.store.listKeywordRules(guildId)
    const configRules = this.deps.config.moderation.keywordRules
      .filter((rule) => rule.guildId === guildId || rule.guildId === '*')
      .map((rule) => convertKeywordRuleConfig(rule))
    return mergeRules(storedRules, configRules)
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

function convertKeywordRuleConfig(rule: StuhelperKeywordRuleConfig) {
  const now = new Date(0)
  return {
    ...rule,
    enabled: rule.enabled ?? true,
    note: rule.note || null,
    createdAt: now,
    updatedAt: now,
  }
}

function mergeRules(storedRules: Awaited<ReturnType<ModerationStore['listKeywordRules']>>, configRules: ReturnType<typeof convertKeywordRuleConfig>[]) {
  const merged = new Map<string, ReturnType<typeof convertKeywordRuleConfig>>()
  for (const rule of storedRules) {
    merged.set(rule.id, rule)
  }
  for (const rule of configRules) {
    if (!merged.has(rule.id)) {
      merged.set(rule.id, rule)
    }
  }
  return [...merged.values()]
}
