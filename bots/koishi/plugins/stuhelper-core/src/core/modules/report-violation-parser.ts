import type { ReporterPenalty, ViolationInfo } from './report-types'
import { ViolationLevel } from './report-types'

const MAX_REASON_LENGTH = 500
const MAX_BAN_SECONDS = 604_800
const MAX_WARN_COUNT = 5
const MAX_REPORTER_LIMIT_MINUTES = 1_440

export function parseViolationInfo(response: string): ViolationInfo {
  const jsonText = response.startsWith('{') && response.endsWith('}')
    ? response
    : response.match(/\{[\s\S]*\}/g)?.[0]
  if (!jsonText) throw new Error('无法解析AI响应中的JSON')

  return parseViolationPayload(JSON.parse(jsonText))
}

function parseViolationPayload(value: unknown): ViolationInfo {
  if (!isRecord(value)) throw new Error('AI响应格式不正确')
  const level = parseViolationLevel(value.level)
  const reason = parseReason(value.reason)
  const action = parseViolationActions(value.action, level)
  const reporterPenalty = parseReporterPenalty(value.reporterPenalty)
  return reporterPenalty ? { level, reason, action, reporterPenalty } : { level, reason, action }
}

function parseViolationLevel(value: unknown): ViolationLevel {
  if (typeof value !== 'number' || !Number.isInteger(value) || value < ViolationLevel.NONE || value > ViolationLevel.CRITICAL) {
    throw new Error('AI响应 level 必须是 0 到 4 的整数')
  }
  return value as ViolationLevel
}

function parseReason(value: unknown): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error('AI响应 reason 必须是非空字符串')
  }
  if (value.length > MAX_REASON_LENGTH) {
    throw new Error(`AI响应 reason 不能超过 ${MAX_REASON_LENGTH} 个字符`)
  }
  return value
}

function parseViolationActions(value: unknown, level: ViolationLevel): ViolationInfo['action'] {
  if (!Array.isArray(value)) throw new Error('AI响应 action 必须是数组')
  if (level === ViolationLevel.NONE && value.length > 0) {
    throw new Error('AI响应 level 为 0 时 action 必须为空')
  }
  return value.map(parseViolationAction)
}

function parseViolationAction(value: unknown): ViolationInfo['action'][number] {
  if (!isRecord(value) || typeof value.type !== 'string') {
    throw new Error('AI响应 action.type 必须是字符串')
  }
  if (value.type === 'ban') return { type: 'ban', time: parseBanTime(value.time) }
  if (value.type === 'warn') return { type: 'warn', count: parseWarnCount(value.count) }
  if (value.type === 'kick') return { type: 'kick' }
  if (value.type === 'kick_blacklist') return { type: 'kick_blacklist' }
  throw new Error(`AI响应 action.type 不支持: ${value.type}`)
}

function parseBanTime(value: unknown): number {
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0 || value > MAX_BAN_SECONDS) {
    throw new Error(`AI响应 ban time 必须是 1 到 ${MAX_BAN_SECONDS} 秒的整数`)
  }
  return value
}

function parseWarnCount(value: unknown): number {
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0 || value > MAX_WARN_COUNT) {
    throw new Error(`AI响应 warn count 必须是 1 到 ${MAX_WARN_COUNT} 的整数`)
  }
  return value
}

function parseReporterPenalty(value: unknown): ReporterPenalty | undefined {
  if (typeof value === 'undefined') return undefined
  if (!isRecord(value) || typeof value.shouldLimit !== 'boolean') {
    throw new Error('AI响应 reporterPenalty.shouldLimit 必须是布尔值')
  }
  return {
    shouldLimit: value.shouldLimit,
    duration: parseOptionalReporterDuration(value.duration),
    reason: parseOptionalPenaltyReason(value.reason),
  }
}

function parseOptionalReporterDuration(value: unknown): number | undefined {
  if (typeof value === 'undefined') return undefined
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0 || value > MAX_REPORTER_LIMIT_MINUTES) {
    throw new Error(`AI响应 reporterPenalty.duration 必须是 1 到 ${MAX_REPORTER_LIMIT_MINUTES} 分钟的整数`)
  }
  return value
}

function parseOptionalPenaltyReason(value: unknown): string | undefined {
  if (typeof value === 'undefined') return undefined
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error('AI响应 reporterPenalty.reason 必须是非空字符串')
  }
  return value
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
