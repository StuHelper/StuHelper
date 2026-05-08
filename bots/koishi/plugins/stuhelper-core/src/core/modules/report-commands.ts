import { h, Logger } from 'koishi'

import { buildReportPrompt } from './report-context'
import type { ReportModule } from './report.module'
import type { ReporterPenalty, ViolationInfo } from './report-types'
import { ViolationLevel } from './report-types'
import { parseViolationInfo } from './report-violation-parser'

export { parseViolationInfo } from './report-violation-parser'

const logger = new Logger('stuhelperGroupCenter:report')
const DEFAULT_AUTHORITY = 1
const DEFAULT_REPORTER_LIMIT_MINUTES = 60
const SECONDS_PER_MINUTE = 60
const MS_PER_SECOND = 1000
const MS_PER_MINUTE = SECONDS_PER_MINUTE * MS_PER_SECOND

interface ReportCommandInput {
  readonly host: ReportModule
  readonly session: any
  readonly options: any
}

interface ReportTarget {
  readonly quoteId: string
  readonly messageReportKey: string
  readonly reportedUserId: string
  readonly content: string
  readonly message: any
}

export function registerReportCommands(host: ReportModule): void {
  host.registerCommand({
    name: 'report',
    desc: '举报违规消息',
    permNode: 'report',
    permDesc: '使用举报功能',
    skipAuth: true,
    usage: '回复违规消息使用，AI自动审核处理',
  })
    .option('verbose', '-v 显示详细判断结果', { fallback: true })
    .action(async ({ session, options }) => handleReportCommand({ host, session, options }))
}

async function handleReportCommand(input: ReportCommandInput): Promise<string> {
  const initialError = validateReportCommand(input)
  if (initialError) return initialError

  const userAuthority = await getUserAuthority(input)
  const cooldownMessage = getCooldownMessage(input, userAuthority)
  if (cooldownMessage) return cooldownMessage
  if (!input.session.quote) return quote(input.session) + '请回复需要举报的消息。例如：回复某消息 > /report'

  try {
    return await processReport({ ...input, userAuthority })
  } catch (error: any) {
    return handleReportFailure({ ...input, userAuthority, error })
  }
}

function validateReportCommand(input: ReportCommandInput): string | null {
  if (!input.host.config.report?.enabled) return quote(input.session) + '举报功能已被禁用'
  if (!input.session.guildId) return quote(input.session) + '此命令只能在群聊中使用。'

  const guildConfig = input.host.getReportGuildConfig(input.session.guildId)
  if (guildConfig && !guildConfig.enabled) return quote(input.session) + '本群的举报功能已被禁用'
  return null
}

async function processReport(input: ReportCommandInput & {
  readonly userAuthority: number
}): Promise<string> {
  const target = await loadReportTarget(input)
  if (typeof target === 'string') return target

  await input.host.logCommand({
    session: input.session,
    command: 'report',
    target: target.reportedUserId,
    details: `举报内容: ${target.content}`,
  })
  const timeError = validateReportTime({ ...input, target })
  if (timeError) return timeError

  const violationInfo = await moderateReportTarget({ ...input, target })
  if (typeof violationInfo === 'string') return violationInfo

  const result = await input.host.handleViolation({
    session: input.session,
    userId: target.reportedUserId,
    violation: violationInfo,
    content: target.content,
    verbose: input.options.verbose,
    guildConfig: input.host.getReportGuildConfig(input.session.guildId),
  })

  recordReportedMessage(input.host, target, violationInfo)
  return await formatReportResult({ ...input, violationInfo, result })
}

async function getUserAuthority(input: ReportCommandInput): Promise<number> {
  try {
    if (!input.host.ctx.database) return DEFAULT_AUTHORITY
    const user = await input.host.ctx.database.getUser(input.session.platform, input.session.userId)
    return user?.authority || DEFAULT_AUTHORITY
  } catch (error) {
    logger.error('获取用户权限失败:', error)
    return DEFAULT_AUTHORITY
  }
}

function getCooldownMessage(input: ReportCommandInput, userAuthority: number): string | null {
  if (userAuthority >= input.host.getMinUnlimitedAuthority()) return null

  const banKey = `${input.session.userId}:${input.session.guildId}`
  const banRecord = input.host.reportBans[banKey]
  if (!banRecord || Date.now() >= banRecord.expireTime) return null

  const remainingMinutes = Math.ceil((banRecord.expireTime - Date.now()) / MS_PER_MINUTE)
  return quote(input.session) + `您由于举报不当已被暂时限制使用举报功能，请在${remainingMinutes}分钟后再试。`
}

async function loadReportTarget(input: ReportCommandInput): Promise<ReportTarget | string> {
  const quoteId = typeof input.session.quote === 'string'
    ? input.session.quote
    : input.session.quote.id || input.session.quote.messageId
  if (!quoteId) return quote(input.session) + '无法读取被举报消息的 ID。'

  const messageReportKey = `${input.session.guildId}:${quoteId}`
  const reportedRecord = input.host.reportedMessages[messageReportKey]
  if (reportedRecord) return quote(input.session) + `该消息已被举报过，处理结果: ${reportedRecord.result}`

  const reportedMessage = await input.session.bot.getMessage(input.session.guildId, quoteId)
  if (!reportedMessage || !reportedMessage.content) return quote(input.session) + '无法获取被举报的消息内容。'

  const reportedUserId = resolveReportedUserId(reportedMessage)
  if (reportedUserId === null) return '无法确定被举报消息的发送者。'
  if (!reportedUserId) return quote(input.session) + '无法确定被举报消息的发送者。'
  if (reportedUserId === input.session.userId) return quote(input.session) + '不能举报自己的消息喵~'
  if (reportedUserId === input.session.selfId) return quote(input.session) + '喵？不能举报本喵的消息啦~'

  return { quoteId, messageReportKey, reportedUserId, content: reportedMessage.content, message: reportedMessage }
}

function resolveReportedUserId(reportedMessage: any): string | null {
  if (reportedMessage.user && typeof reportedMessage.user === 'object') return reportedMessage.user.id
  if (typeof reportedMessage.userId === 'string') return reportedMessage.userId

  const sender = reportedMessage.sender || reportedMessage.from
  if (sender && typeof sender === 'object' && sender.id) return sender.id
  return null
}

function validateReportTime(input: ReportCommandInput & {
  readonly userAuthority: number
  readonly target: ReportTarget
}): string | null {
  if (input.userAuthority >= input.host.getMinUnlimitedAuthority()) return null

  const messageTimestamp = extractMessageTimestamp(input.target.message)
  const maxReportTimeMs = input.host.getMaxReportTime() * MS_PER_MINUTE
  if (messageTimestamp > 0 && Date.now() - messageTimestamp > maxReportTimeMs) {
    return quote(input.session) + `只能举报${input.host.getMaxReportTime()}分钟内的消息，此消息已超时。`
  }
  return null
}

function extractMessageTimestamp(message: any): number {
  if (message.timestamp) return message.timestamp
  if (typeof message.time === 'number') return message.time
  if (!message.date) return 0

  if (message.date instanceof Date) return message.date.getTime()
  if (typeof message.date === 'string') return new Date(message.date).getTime()
  if (typeof message.date === 'number') return message.date
  return 0
}

async function moderateReportTarget(input: ReportCommandInput & {
  readonly userAuthority: number
  readonly target: ReportTarget
}): Promise<ViolationInfo | string> {
  const guildConfig = input.host.getReportGuildConfig(input.session.guildId)
  const prompt = buildReportPrompt({
    host: input.host,
    guildId: input.session.guildId,
    content: input.target.content,
    guildConfig,
  })
  const response = await input.host.callModeration(prompt)
  return parseViolationInfoOrLimitReporter({ ...input, response })
}

async function parseViolationInfoOrLimitReporter(input: ReportCommandInput & {
  readonly userAuthority: number
  readonly response: string
}): Promise<ViolationInfo | string> {
  try {
    return parseViolationInfo(input.response)
  } catch (error) {
    logger.error('解析AI响应失败:', error, input.response)
    if (input.userAuthority < input.host.getMinUnlimitedAuthority()) {
      await limitReporter({ ...input, logResult: '举报处理失败，已限制使用' })
    }
    return quote(input.session) + '举报处理失败：AI判断结果格式有误，请重试或联系管理员手动处理。'
  }
}

function recordReportedMessage(
  host: ReportModule,
  target: ReportTarget,
  violationInfo: ViolationInfo,
): void {
  host.reportedMessages[target.messageReportKey] = {
    messageId: target.quoteId,
    timestamp: Date.now(),
    result: violationInfo.level > ViolationLevel.NONE
      ? `已处理(${host.getViolationLevelText(violationInfo.level)}违规)`
      : '未违规',
  }
}

async function formatReportResult(input: ReportCommandInput & {
  readonly userAuthority: number
  readonly violationInfo: ViolationInfo
  readonly result: string
}): Promise<string> {
  const penalty = input.violationInfo.reporterPenalty
  if (!shouldLimitReporter(input, penalty)) return quote(input.session) + input.result

  const reason = penalty?.reason || '滥用举报功能'
  const durationMinutes = penalty?.duration || DEFAULT_REPORTER_LIMIT_MINUTES
  await limitReporter({
    ...input,
    durationMs: durationMinutes * MS_PER_MINUTE,
    logResult: `AI判定: ${reason}，限制${penalty?.duration}分钟`,
  })
  return quote(input.session) + input.result +
    `\nAI判断理由：${input.violationInfo.reason}\n您因${reason}，已被暂时限制举报功能${penalty?.duration}分钟。`
}

function shouldLimitReporter(input: ReportCommandInput & {
  readonly userAuthority: number
}, penalty?: ReporterPenalty): boolean {
  return !!penalty?.shouldLimit && input.userAuthority < input.host.getMinUnlimitedAuthority()
}

async function handleReportFailure(input: ReportCommandInput & {
  readonly userAuthority: number
  readonly error: any
}): Promise<string> {
  logger.error('举报处理失败:', input.error)
  if (input.userAuthority < input.host.getMinUnlimitedAuthority()) {
    await limitReporter({
      ...input,
      logResult: `举报处理失败(${input.error.message})，已限制使用`,
    })
  }
  return quote(input.session) + `举报处理失败：${input.error.message}`
}

async function limitReporter(input: ReportCommandInput & {
  readonly logResult: string
  readonly durationMs?: number
}): Promise<void> {
  const duration = input.durationMs ?? input.host.getReportCooldownDuration()
  const banKey = `${input.session.userId}:${input.session.guildId}`
  input.host.reportBans[banKey] = {
    userId: input.session.userId,
    guildId: input.session.guildId,
    timestamp: Date.now(),
    expireTime: Date.now() + duration,
  }
  await input.host.logCommand({
    session: input.session,
    command: 'report-banned',
    target: input.session.userId,
    details: input.logResult,
  })
}

function quote(session: any): string {
  return h.quote(session.messageId) as unknown as string
}
