import { Logger } from 'koishi'
import { createPlatformClient } from '@stuhelper/koishi-shared'

import type { ReportModule } from './report.module'
import type { ViolationAction, ViolationInfo } from './report-types'
import { ViolationLevel } from './report-types'
import { formatDurationSeconds, shorten } from './report-format'

const logger = new Logger('stuhelperGroupCenter:report')
const SHORT_CONTENT_MAX_LENGTH = 30
const ERROR_MESSAGE_MAX_LENGTH = 50
const MS_PER_SECOND = 1000

export interface ReportViolationInput {
  readonly host: ReportModule
  readonly session: any
  readonly userId: string
  readonly violation: ViolationInfo
  readonly content: string
  readonly verbose: boolean
  readonly guildConfig: any
}

export async function handleReportViolation(input: ReportViolationInput): Promise<string> {
  try {
    return await executeViolation(input)
  } catch (error: any) {
    return handleViolationFailure(input, error)
  }
}

export function getViolationLevelText(level: ViolationLevel): string {
  switch (level) {
    case ViolationLevel.NONE: return '未'
    case ViolationLevel.LOW: return '轻微'
    case ViolationLevel.MEDIUM: return '中度'
    case ViolationLevel.HIGH: return '严重'
    case ViolationLevel.CRITICAL: return '极其严重'
    default: return '未知'
  }
}

async function executeViolation(input: ReportViolationInput): Promise<string> {
  if (input.violation.level === ViolationLevel.NONE) return formatNoViolation(input)

  const shouldAutoProcess = input.guildConfig
    ? input.guildConfig.autoProcess
    : input.host.config.report?.autoProcess
  if (!shouldAutoProcess) return handleManualViolation(input)

  const actionResults: string[] = []
  for (const action of input.violation.action || []) {
    await executeReportAction({ ...input, action, actionResults })
  }

  const result = formatProcessedViolation(input, actionResults)
  await logReportHandling(input, actionResults).catch((error) => {
    logger.error('记录举报处理日志失败:', error)
  })
  return result
}

function formatNoViolation(input: ReportViolationInput): string {
  return input.verbose
    ? `AI判断结果：该消息未违规\n理由：${input.violation.reason}`
    : '该消息未被判定为违规内容。'
}

async function handleManualViolation(input: ReportViolationInput): Promise<string> {
  const levelText = getViolationLevelText(input.violation.level)
  await input.host.logCommand({
    session: input.session,
    command: 'report-no-action',
    target: input.userId,
    details: `${levelText}违规，管理员待处理`,
  })
  return input.verbose
    ? `AI判断结果：${levelText}违规\n理由：${input.violation.reason}\n操作：自动处理功能已禁用，请管理员手动处理`
    : `该消息被判定为${levelText}违规，请管理员手动处理。`
}

function formatProcessedViolation(input: ReportViolationInput, actionResults: string[]): string {
  const levelText = getViolationLevelText(input.violation.level)
  if ((input.violation.action || []).length === 0) {
    return input.verbose
      ? `AI判断结果：${levelText}违规\n理由：${input.violation.reason}\n操作：无需处理`
      : `该消息被判定为${levelText}违规，无需处理。`
  }

  const actionText = actionResults.join('、')
  return input.verbose
    ? `AI判断结果：${levelText}违规\n理由：${input.violation.reason}\n操作：${actionText}`
    : `已对用户 ${input.userId} 执行：${actionText}，${levelText}违规。`
}

async function logReportHandling(input: ReportViolationInput, actionResults: string[]): Promise<void> {
  const levelText = getViolationLevelText(input.violation.level)
  const actionText = formatActionText(input.violation.action)
  const shortContent = shorten(input.content, SHORT_CONTENT_MAX_LENGTH)
  await input.host.logCommand({
    session: input.session,
    command: 'report-handle',
    target: input.userId,
    details: `${levelText}违规，处理: ${actionText}，内容: ${shortContent}`,
  })
  await input.host.ctx.stuhelperGroupCenter.pushMessage(
    input.session.bot,
    `[举报] 群${input.session.guildId} 用户 ${input.userId} - ${levelText}违规\n内容: ${shortContent}\n处理: ${actionText}`,
    'warning',
  )
}

function formatActionText(actions: ViolationAction[]): string {
  if (actions.length === 0) return '无操作'
  return actions.map(formatAction).join('、')
}

function formatAction(action: ViolationAction): string {
  switch (action.type) {
    case 'ban': return `禁言${action.time}秒`
    case 'warn': return `警告${action.count}次`
    case 'kick': return '踢出群聊'
    case 'kick_blacklist': return '踢出并拉黑'
    default: return action.type
  }
}

async function handleViolationFailure(input: ReportViolationInput, error: any): Promise<string> {
  logger.error('执行违规处理失败:', error)
  await logViolationFailure(input, error).catch((innerError) => {
    logger.error('记录举报错误日志失败:', innerError)
  })
  const levelText = getViolationLevelText(input.violation.level)
  return `AI已判定该消息${levelText}违规，但自动处理失败：${error.message}\n请联系管理员手动处理。`
}

async function logViolationFailure(input: ReportViolationInput, error: any): Promise<void> {
  const levelText = getViolationLevelText(input.violation.level)
  const message = shorten(error.message, ERROR_MESSAGE_MAX_LENGTH)
  await input.host.logCommand({
    session: input.session,
    command: 'report-error',
    target: input.userId,
    details: `${levelText}违规处理失败: ${message}`,
  })
  await input.host.ctx.stuhelperGroupCenter.pushMessage(
    input.session.bot,
    `[举报失败] 用户 ${input.userId} - ${levelText}违规\n错误: ${message}`,
    'warning',
  )
}

async function executeReportAction(input: ReportViolationInput & {
  readonly action: ViolationAction
  readonly actionResults: string[]
}): Promise<void> {
  try {
    await runReportAction(input)
  } catch (error) {
    logger.error(`执行操作失败: ${input.action.type}`, error)
    input.actionResults.push(`${input.action.type}操作失败`)
  }
}

async function runReportAction(input: ReportViolationInput & {
  readonly action: ViolationAction
  readonly actionResults: string[]
}): Promise<void> {
  const { action, actionResults, session, userId } = input
  if (action.type === 'ban' && action.time && action.time > 0) {
    await banUserBySeconds({ host: input.host, session, userId, seconds: action.time })
    actionResults.push(`禁言${action.time}秒`)
  } else if (action.type === 'warn' && action.count && action.count > 0) {
    await warnUser({ host: input.host, session, userId, count: action.count })
    actionResults.push(`警告${action.count}次`)
  } else if (action.type === 'kick') {
    await kickUser({ host: input.host, session, userId, addToBlacklist: false })
    actionResults.push('踢出群聊')
  } else if (action.type === 'kick_blacklist') {
    await kickUser({ host: input.host, session, userId, addToBlacklist: true })
    actionResults.push('踢出群聊并加入黑名单')
  } else if (action.type !== 'ban' && action.type !== 'warn') {
    logger.warn(`未知的操作类型: ${action.type}`)
  }
}

async function warnUser(input: {
  readonly host: ReportModule
  readonly session: any
  readonly userId: string
  readonly count: number
}): Promise<void> {
  const guildWarns = input.host.data.warns.get(input.session.guildId) || {}
  const current = guildWarns[input.userId] || { count: 0, timestamp: 0 }
  guildWarns[input.userId] = { count: current.count + input.count, timestamp: Date.now() }
  input.host.data.warns.set(input.session.guildId, guildWarns)
  input.host.data.warns.flush()
  await input.host.logCommand({
    session: input.session,
    command: 'report-warn',
    target: input.userId,
    details: `AI 处置警告 ${input.count} 次`,
  })
}

async function banUserBySeconds(input: {
  readonly host: ReportModule
  readonly session: any
  readonly userId: string
  readonly seconds: number
}): Promise<void> {
  try {
    const milliseconds = input.seconds * MS_PER_SECOND
    await input.session.bot.muteGuildMember(input.session.guildId, input.userId, milliseconds)
    recordReportMute({
      host: input.host,
      guildId: input.session.guildId,
      userId: input.userId,
      duration: milliseconds,
    })
    await input.host.logCommand({
      session: input.session,
      command: 'report-ban',
      target: input.userId,
      details: `AI 处置禁言 ${formatDurationSeconds(input.seconds)}`,
    })
  } catch (error: any) {
    logger.error(`按秒数禁言用户失败: ${error.message}`)
    throw error
  }
}

async function kickUser(input: {
  readonly host: ReportModule
  readonly session: any
  readonly userId: string
  readonly addToBlacklist: boolean
}): Promise<void> {
  try {
    await input.session.bot.kickGuildMember(input.session.guildId, input.userId, false)
    if (input.addToBlacklist) {
      await createModerationBlacklist(input)
    }
    await input.host.logCommand({
      session: input.session,
      command: 'report-kick',
      target: input.userId,
      details: 'AI 处置踢出群聊',
    })
  } catch (error: any) {
    logger.error(`踢出用户失败: ${error.message}`)
    throw error
  }
}

async function createModerationBlacklist(input: {
  readonly host: ReportModule
  readonly session: any
  readonly userId: string
}) {
  await createPlatformClient(input.host.platformConfig).createMemberBlacklist({
    platform: input.session.platform,
    subjectType: 'qq_user',
    subjectID: input.userId,
    scopeType: 'guild',
    guildID: input.session.guildId,
    source: 'moderation_action',
    reasonCode: 'violation_review_blacklist',
    reasonText: 'AI moderation action',
    createdFrom: 'moderation_review',
    operatorID: input.session.userId,
    metadata: { rawCommand: input.session.content || '' },
  })
}

function recordReportMute(input: {
  readonly host: ReportModule
  readonly guildId: string
  readonly userId: string
  readonly duration: number
}): void {
  const { host, guildId, userId, duration } = input
  const guildMutes = host.data.mutes.get(guildId) || {}
  guildMutes[userId] = { startTime: Date.now(), duration, remainingTime: duration }
  host.data.mutes.set(guildId, guildMutes)
  host.data.mutes.flush()
}
