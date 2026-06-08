import type { Logger, Session, Universal } from 'koishi'

import { ModerationActionService, type ModerationStore } from '@stuhelper/koishi-moderation-core'
import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type StuhelperAIConfig,
  type StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

const AI_REVIEW_TIMEOUT_MS = 10_000

interface AIReviewResult {
  severity: 'none' | 'low' | 'medium' | 'high'
  summary: string
}

interface ReportServiceDeps {
  store: ModerationStore
  actions: ModerationActionService
  logger: Logger
  config: StuhelperGroupGuardPluginConfig
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
    const messages = resolveGroupGuardMessages(this.deps.config.messages)
    if (!guildId || !channelId) {
      return renderMessageTemplate(messages.reportGroupOnly)
    }

    const input = { session, guildId, channelId, targetMemberId, reason }
    const report = await this.createReport(input)

    if (!this.deps.config.ai.enabled) {
      return renderMessageTemplate(messages.reportRecordedAIUnavailable)
    }
    return this.reviewReportWithAI(input, report.id)
  }

  private async createReport(input: ReportRuntimeInput) {
    const report = await this.deps.store.createReport({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      reporterMemberId: input.session.userId,
      targetMemberId: input.targetMemberId,
      reason: input.reason,
      aiStatus: this.deps.config.ai.enabled ? 'pending' : 'disabled',
      aiSeverity: 'none',
      aiSummary: null,
    })
    await this.appendReportCreatedEvent(input, report.id)
    return report
  }

  private async appendReportCreatedEvent(input: ReportRuntimeInput, reportId: string) {
    await this.deps.store.appendEvent({
      platform: input.session.platform,
      botSelfId: input.session.selfId,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      type: 'report_created',
      level: 'medium',
      summary: `${input.session.userId} 举报了 ${input.targetMemberId}`,
      payload: { reason: input.reason, reportID: reportId },
    })
  }

  private async reviewReportWithAI(input: ReportRuntimeInput, reportId: string) {
    try {
      const result = await reviewReportWithAI(this.deps.config.ai, {
        guildId: input.guildId,
        reporterMemberId: input.session.userId,
        targetMemberId: input.targetMemberId,
        reason: input.reason,
      })
      await this.deps.store.updateReportAIResult({
        id: reportId,
        aiStatus: 'completed',
        aiSeverity: result.severity,
        aiSummary: result.summary,
      })
      await this.appendAIReviewedEvent({ input, result, reportId })
      return this.applyAIResult({
        bot: input.session.bot,
        guildId: input.guildId,
        channelId: input.channelId,
        targetMemberId: input.targetMemberId,
        reason: input.reason,
        result,
      })
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
      return renderMessageTemplate(resolveGroupGuardMessages(this.deps.config.messages).reportAIReviewFailed)
    }
  }

  private async appendAIReviewedEvent(input: {
    readonly input: ReportRuntimeInput
    readonly result: AIReviewResult
    readonly reportId: string
  }) {
    await this.deps.store.appendEvent({
      platform: input.input.session.platform,
      botSelfId: input.input.session.selfId,
      guildId: input.input.guildId,
      channelId: input.input.channelId,
      memberId: input.input.targetMemberId,
      type: 'report_ai_reviewed',
      level: input.result.severity === 'high' ? 'critical' : input.result.severity === 'medium' ? 'high' : 'low',
      summary: `AI 已完成举报审核：${input.result.summary}`,
      payload: { severity: input.result.severity, reportID: input.reportId },
    })
  }

  private async applyAIResult(input: ApplyAIResultInput) {
    const { result } = input
    if (result.severity === 'high') {
      await this.createHighRiskReview(input)
      return renderMessageTemplate(resolveGroupGuardMessages(this.deps.config.messages).reportHighRisk)
    }

    if (result.severity === 'medium') {
      await this.warnAIReport(input)
      await this.muteAIReport(input)
      return renderMessageTemplate(resolveGroupGuardMessages(this.deps.config.messages).reportMediumRisk)
    }

    if (result.severity === 'low') {
      await this.warnAIReport(input)
      return renderMessageTemplate(resolveGroupGuardMessages(this.deps.config.messages).reportLowRisk)
    }

    return renderMessageTemplate(resolveGroupGuardMessages(this.deps.config.messages).reportNoAction)
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

  private async warnAIReport(input: ApplyAIResultInput) {
    await this.deps.actions.warnMember({
      runtime: { platform: input.bot.platform || '', botSelfId: input.bot.selfId },
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      reason: `AI 举报审核：${input.reason}`,
    })
  }

  private async muteAIReport(input: ApplyAIResultInput) {
    await this.deps.actions.muteMember({
      bot: input.bot,
      guildId: input.guildId,
      channelId: input.channelId,
      memberId: input.targetMemberId,
      seconds: this.deps.config.moderation.defaultMuteSeconds,
      reason: 'AI 举报审核命中中风险',
    })
  }
}

async function reviewReportWithAI(config: StuhelperAIConfig, input: Record<string, string>): Promise<AIReviewResult> {
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
    summary: payload.summary || '未返回摘要',
  }
}
