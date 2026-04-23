import type { Logger, Session, Universal } from 'koishi'

import { ModerationActionService, type ModerationStore } from '@stuhelper/koishi-moderation-core'
import type { StuhelperAIConfig, StuhelperGroupGuardPluginConfig } from '@stuhelper/koishi-shared'

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

export class ReportService {
  constructor(private readonly deps: ReportServiceDeps) {}

  async handleReport(session: Session, targetMemberId: string, reason: string) {
    const guildId = session.guildId
    const channelId = session.channelId
    if (!guildId || !channelId) {
      return '举报命令只能在群聊中使用。'
    }

    const report = await this.deps.store.createReport({
      platform: session.platform,
      botSelfId: session.selfId,
      guildId,
      channelId,
      reporterMemberId: session.userId,
      targetMemberId,
      reason,
      aiStatus: this.deps.config.ai.enabled ? 'pending' : 'disabled',
      aiSeverity: 'none',
      aiSummary: null,
    })
    await this.deps.store.appendEvent({
      platform: session.platform,
      botSelfId: session.selfId,
      guildId,
      channelId,
      memberId: targetMemberId,
      type: 'report_created',
      level: 'medium',
      summary: `${session.userId} 举报了 ${targetMemberId}`,
      payload: { reason, reportID: report.id },
    })

    if (!this.deps.config.ai.enabled) {
      return '举报已记录。当前未启用 AI 审核，事件已进入人工处理范围。'
    }

    try {
      const result = await reviewReportWithAI(this.deps.config.ai, {
        guildId,
        reporterMemberId: session.userId,
        targetMemberId,
        reason,
      })
      await this.deps.store.updateReportAIResult(report.id, 'completed', result.severity, result.summary)
      await this.deps.store.appendEvent({
        platform: session.platform,
        botSelfId: session.selfId,
        guildId,
        channelId,
        memberId: targetMemberId,
        type: 'report_ai_reviewed',
        level: result.severity === 'high' ? 'critical' : result.severity === 'medium' ? 'high' : 'low',
        summary: `AI 已完成举报审核：${result.summary}`,
        payload: { severity: result.severity, reportID: report.id },
      })
      return this.applyAIResult(session.bot, guildId, channelId, targetMemberId, reason, result)
    } catch (error) {
      this.deps.logger.warn('report ai review failed', {
        targetMemberId,
        error: error instanceof Error ? error.message : String(error),
      })
      await this.deps.store.updateReportAIResult(report.id, 'failed', 'none', null)
      return '举报已记录，但 AI 审核失败，事件已保留供人工处理。'
    }
  }

  private async applyAIResult(bot: ReportBot, guildId: string, channelId: string, targetMemberId: string, reason: string, result: AIReviewResult) {
    if (result.severity === 'high') {
      await this.deps.store.createReview({
        platform: bot.platform || '',
        botSelfId: bot.selfId,
        guildId,
        channelId,
        memberId: targetMemberId,
        actionType: 'kick_and_block',
        status: 'pending',
        reason,
        operatorMemberId: null,
        resolutionNote: null,
        payload: { source: 'ai-report', summary: result.summary },
      })
      return '举报已提交，AI 判定为高风险，已进入踢人/拉黑人工复核队列。'
    }

    if (result.severity === 'medium') {
      await this.deps.actions.warnMember({
        platform: bot.platform || '',
        botSelfId: bot.selfId,
      }, guildId, channelId, targetMemberId, `AI 举报审核：${reason}`)
      await this.deps.actions.muteMember(bot, guildId, channelId, targetMemberId, this.deps.config.moderation.defaultMuteSeconds, 'AI 举报审核命中中风险')
      return '举报已提交，AI 判定为中风险，已自动警告并禁言。'
    }

    if (result.severity === 'low') {
      await this.deps.actions.warnMember({
        platform: bot.platform || '',
        botSelfId: bot.selfId,
      }, guildId, channelId, targetMemberId, `AI 举报审核：${reason}`)
      return '举报已提交，AI 判定为低风险，已自动记警告。'
    }

    return '举报已提交，AI 未判定出可执行违规动作。'
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
