import type { Logger, Session, Universal } from 'koishi'

import { ModerationActionService, type ModerationStore } from '@stuhelper/koishi-moderation-core'
import {
  DEFAULT_GROUP_GUARD_MODERATION_SETTINGS,
  type GroupGuardAISettings,
  type GroupGuardBehaviorSettingsStore,
} from '@stuhelper/koishi-shared'

import {
  getGroupGuardAISettings,
  type GroupGuardAISettingsProvider,
} from './group-guard-ai-provider'
import {
  getGroupGuardMessages,
  groupGuardMessage,
  type GroupGuardMessageProvider,
  type GroupGuardMessages,
} from './group-guard-message-provider'

const AI_REVIEW_TIMEOUT_MS = 10_000

interface AIReviewResult {
  severity: 'none' | 'low' | 'medium' | 'high'
  summary: string
}

interface ReportServiceDeps {
  store: ModerationStore
  actions: ModerationActionService
  logger: Logger
  aiSettings?: GroupGuardAISettingsProvider
  behaviorSettings?: GroupGuardBehaviorSettingsStore
  messageProvider?: GroupGuardMessageProvider
}

type ReportBot = Universal.Methods & { platform?: string, selfId: string }

interface ReportRuntimeInput {
  readonly session: Session
  readonly guildId: string
  readonly channelId: string
  readonly targetMemberId: string
  readonly reason: string
}

interface ApplyAIResultInput {
  readonly bot: ReportBot
  readonly guildId: string
  readonly channelId: string
  readonly targetMemberId: string
  readonly reason: string
  readonly result: AIReviewResult
}

export class ReportService {
  constructor(private readonly deps: ReportServiceDeps) {}

  async handleReport(session: Session, targetMemberId: string, reason: string) {
    const guildId = session.guildId
    const channelId = session.channelId
    const messages = await this.getMessages()
    if (!guildId || !channelId) {
      return groupGuardMessage(messages, 'reportGroupOnly')
    }

    const input = { session, guildId, channelId, targetMemberId, reason }
    const ai = await this.getAISettings()
    const report = await this.createReport(input, messages, ai)

    if (!ai.enabled) {
      return groupGuardMessage(messages, 'reportRecordedAIUnavailable')
    }
    return this.reviewReportWithAI(input, report.id, messages, ai)
  }

  private async createReport(
    input: ReportRuntimeInput,
    messages: GroupGuardMessages,
    ai: GroupGuardAISettings,
  ) {
    const report = await this.deps.store.createReport({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      reporterMemberId: input.session.userId,
      targetMemberId: input.targetMemberId,
      reason: input.reason,
      aiStatus: ai.enabled ? 'pending' : 'disabled',
      aiSeverity: 'none',
      aiSummary: null,
    })
    await this.appendReportCreatedEvent(input, report.id, messages)
    return report
  }

  private async appendReportCreatedEvent(
    input: ReportRuntimeInput,
    reportId: string,
    messages: GroupGuardMessages,
  ) {
    await this.deps.store.appendEvent({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      type: 'report_created',
      level: 'medium',
      summary: groupGuardMessage(messages, 'reportCreatedEventSummary', {
        reporterMemberId: input.session.userId,
        targetMemberId: input.targetMemberId,
      }),
      payload: { reason: input.reason, reportID: reportId },
    })
  }

  private async reviewReportWithAI(
    input: ReportRuntimeInput,
    reportId: string,
    messages: GroupGuardMessages,
    ai: GroupGuardAISettings,
  ) {
    try {
      const result = await reviewReportWithAI(ai, {
        guildId: input.guildId,
        reporterMemberId: input.session.userId,
        targetMemberId: input.targetMemberId,
        reason: input.reason,
      }, groupGuardMessage(messages, 'reportAISummaryFallback'))
      await this.deps.store.updateReportAIResult({
        id: reportId,
        aiStatus: 'completed',
        aiSeverity: result.severity,
        aiSummary: result.summary,
      })
      await this.appendAIReviewedEvent({ input, result, reportId, messages })
      return this.applyAIResult({
        bot: input.session.bot,
        guildId: input.guildId,
        channelId: input.channelId,
        targetMemberId: input.targetMemberId,
        reason: input.reason,
        result,
      }, messages)
    } catch (error) {
      this.deps.logger.warn('report ai review failed', {
        targetMemberId: input.targetMemberId,
        error: error instanceof Error ? error.message : String(error),
      })
      await this.deps.store.updateReportAIResult({
        id: reportId,
        aiStatus: 'failed',
        aiSeverity: 'none',
        aiSummary: null,
      })
      return groupGuardMessage(messages, 'reportAIReviewFailed')
    }
  }

  private async appendAIReviewedEvent(input: {
    readonly input: ReportRuntimeInput
    readonly result: AIReviewResult
    readonly reportId: string
    readonly messages: GroupGuardMessages
  }) {
    await this.deps.store.appendEvent({
      platform: input.input.session.platform,
      botSelfId: input.input.session.selfId,
      guildId: input.input.guildId,
      channelId: input.input.channelId,
      memberId: input.input.targetMemberId,
      type: 'report_ai_reviewed',
      level: input.result.severity === 'high' ? 'critical' : input.result.severity === 'medium' ? 'high' : 'low',
      summary: groupGuardMessage(input.messages, 'reportAIReviewedEventSummary', {
        summary: input.result.summary,
      }),
      payload: { severity: input.result.severity, reportID: input.reportId },
    })
  }

  private async applyAIResult(input: ApplyAIResultInput, messages: GroupGuardMessages) {
    const { result } = input
    if (result.severity === 'high') {
      await this.createHighRiskReview(input)
      return groupGuardMessage(messages, 'reportHighRisk')
    }

    if (result.severity === 'medium') {
      await this.warnAIReport(input, messages)
      await this.muteAIReport(input, messages)
      return groupGuardMessage(messages, 'reportMediumRisk')
    }

    if (result.severity === 'low') {
      await this.warnAIReport(input, messages)
      return groupGuardMessage(messages, 'reportLowRisk')
    }

    return groupGuardMessage(messages, 'reportNoAction')
  }

  private async createHighRiskReview(input: ApplyAIResultInput) {
    await this.deps.store.createReview({
      platform: input.bot.platform || '',
      botSelfId: input.bot.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      actionType: 'kick_and_block',
      status: 'pending',
      reason: input.reason,
      operatorMemberId: null,
      resolutionNote: null,
      payload: { source: 'ai-report', summary: input.result.summary },
    })
  }

  private async warnAIReport(input: ApplyAIResultInput, messages: GroupGuardMessages) {
    await this.deps.actions.warnMember({
      runtime: { platform: input.bot.platform || '', botSelfId: input.bot.selfId },
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      reason: groupGuardMessage(messages, 'reportAIWarnReason', { reason: input.reason }),
    })
  }

  private async muteAIReport(input: ApplyAIResultInput, messages: GroupGuardMessages) {
    const moderation = this.deps.behaviorSettings
      ? await this.deps.behaviorSettings.getModerationSettings()
      : DEFAULT_GROUP_GUARD_MODERATION_SETTINGS
    await this.deps.actions.muteMember({
      bot: input.bot,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      seconds: moderation.defaultMuteSeconds,
      reason: groupGuardMessage(messages, 'reportAIMuteReason', { reason: input.reason }),
    })
  }

  private async getMessages() {
    return getGroupGuardMessages(this.deps.messageProvider)
  }

  private async getAISettings() {
    return getGroupGuardAISettings(this.deps.aiSettings)
  }
}

async function reviewReportWithAI(
  config: GroupGuardAISettings,
  input: Record<string, string>,
  summaryFallback: string,
): Promise<AIReviewResult> {
  if (!config.endpoint || !config.apiKey || !config.model) {
    throw new Error('ai review configuration is incomplete')
  }
  const response = await fetch(config.endpoint, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${config.apiKey}`,
      'Content-Type': 'application/json',
    },
    signal: AbortSignal.timeout(AI_REVIEW_TIMEOUT_MS),
    body: JSON.stringify({
      model: config.model,
      input,
    }),
  })
  if (!response.ok) {
    throw new Error(`ai review failed: ${response.status}`)
  }
  const payload = await response.json() as { severity?: AIReviewResult['severity'], summary?: string }
  return {
    severity: payload.severity || 'none',
    summary: payload.summary || summaryFallback,
  }
}
