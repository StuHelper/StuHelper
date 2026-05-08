import { evaluateExpression } from './expression'

const MS_PER_SECOND = 1000
const SECONDS_PER_MINUTE = 60
const MINUTES_PER_HOUR = 60
const HOURS_PER_DAY = 24
const MS_PER_MINUTE = SECONDS_PER_MINUTE * MS_PER_SECOND
const MS_PER_HOUR = MINUTES_PER_HOUR * MS_PER_MINUTE
const MS_PER_DAY = HOURS_PER_DAY * MS_PER_HOUR
const TIME_PATTERN = /(\d+\.?\d*)(d(?:ays?)?|h(?:ours?)?|m(?:ins?)?|s(?:econds?)?)/gi
const SINGLE_TIME_PATTERN = /^(.+?)(d(?:ays?)?|h(?:ours?)?|m(?:ins?)?|s(?:econds?)?)$/i

export const MIN_DURATION = MS_PER_SECOND
export const MAX_DURATION = 29 * MS_PER_DAY + 23 * MS_PER_HOUR + 59 * MS_PER_MINUTE + 59 * MS_PER_SECOND

export function parseTimeString(timeStr: string): number {
  try {
    return parseTimeStringStrict(timeStr)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    throw new Error(`时间解析错误: ${message}`)
  }
}

export function formatDuration(milliseconds: number): string {
  const seconds = Math.floor(milliseconds / MS_PER_SECOND)
  const minutes = Math.floor(seconds / SECONDS_PER_MINUTE)
  const hours = Math.floor(minutes / MINUTES_PER_HOUR)
  const days = Math.floor(hours / HOURS_PER_DAY)
  const parts: string[] = []

  if (days > 0) parts.push(`${days}天`)
  if (hours % HOURS_PER_DAY > 0) parts.push(`${hours % HOURS_PER_DAY}小时`)
  if (minutes % MINUTES_PER_HOUR > 0) parts.push(`${minutes % MINUTES_PER_HOUR}分钟`)
  if (seconds % SECONDS_PER_MINUTE > 0) parts.push(`${seconds % SECONDS_PER_MINUTE}秒`)
  return parts.join('')
}

function parseTimeStringStrict(timeStr: string): number {
  const compact = timeStr?.replace(/\s+/g, '')
  if (!compact) throw new Error('未提供时间')

  const combined = parseCombinedDuration(compact)
  if (combined !== null) return clampDuration(combined)

  const match = compact.match(SINGLE_TIME_PATTERN)
  if (!match) throw new Error(`时间格式错误：${timeStr}`)

  const [, expr, unit] = match
  return clampDuration(parseDurationValue(expr) * unitToMilliseconds(unit))
}

function parseCombinedDuration(input: string): number | null {
  const matches = [...input.matchAll(TIME_PATTERN)]
  if (matches.length <= 1) return null
  const coveredInput = matches.map((match) => match[0]).join('')
  if (coveredInput !== input) throw new Error(`时间格式错误：${input}`)
  return matches.reduce((total, match) => {
    return total + parseFloat(match[1]) * unitToMilliseconds(match[2])
  }, 0)
}

function parseDurationValue(expr: string): number {
  const simpleNumber = parseFloat(expr)
  if (!Number.isNaN(simpleNumber) && expr === simpleNumber.toString()) {
    return simpleNumber
  }

  try {
    return evaluateExpression(expr)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    throw new Error(`表达式解析失败: ${message}`)
  }
}

function unitToMilliseconds(unitText: string): number {
  switch (unitText.toLowerCase().charAt(0)) {
    case 'd': return MS_PER_DAY
    case 'h': return MS_PER_HOUR
    case 'm': return MS_PER_MINUTE
    case 's': return MS_PER_SECOND
    default: throw new Error('未知时间单位')
  }
}

function clampDuration(milliseconds: number): number {
  if (milliseconds < MIN_DURATION) return MIN_DURATION
  if (milliseconds > MAX_DURATION) return MAX_DURATION
  return milliseconds
}
