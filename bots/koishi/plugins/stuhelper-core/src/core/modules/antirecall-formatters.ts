import { Element } from 'koishi'

import type { RecalledMessage } from '../../types'

const DEFAULT_QUERY_LIMIT = 10
const MAX_QUERY_LIMIT = 50
const MIN_CONFIG_VALUE = 1
const MAX_RETENTION_DAYS = 365
const MAX_RECORDS_PER_USER = 1000
const LONG_NUMERIC_ARG_LENGTH = 5

export interface RecallQuery {
  userId: string | null
  targetGuildId?: string
  count: number
}

export interface ConfigUpdateResult {
  updates: any
  messages: string[]
}

export function parseRecallQuery(input: string, defaultGuildId?: string): RecallQuery {
  const args = splitRecallArgs(input)
  const targetUser = args[0]
  let count = DEFAULT_QUERY_LIMIT
  let targetGuildId = defaultGuildId

  if (args[1] && !isNaN(parseInt(args[1]))) {
    count = Math.min(parseInt(args[1]), MAX_QUERY_LIMIT)
  }

  if (args[2] && /^\d+$/.test(args[2])) {
    targetGuildId = args[2]
  } else if (args[1] && /^\d+$/.test(args[1]) && args[1].length > LONG_NUMERIC_ARG_LENGTH) {
    if (!args[2]) {
      targetGuildId = args[1]
      count = DEFAULT_QUERY_LIMIT
    }
  }

  return { userId: parseTargetUserId(targetUser), targetGuildId, count }
}

export function parseConfigUpdates(options: any): ConfigUpdateResult {
  const updates: any = {}
  const messages: string[] = []

  applyEnabledOption(options, updates, messages)
  applyDaysOption(options, updates, messages)
  applyMaxOption(options, updates, messages)

  return { updates, messages }
}

export function formatRecallRecords(
  records: RecalledMessage[],
  userId: string,
  showOriginalTime: boolean,
): string {
  let message = `用户 ${records[0].username} (${userId}) 的撤回记录 (${records.length} 条)\n\n`

  records.forEach((record, index) => {
    const recallTime = new Date(record.recallTime).toLocaleString('zh-CN')
    const sanitizedContent = sanitizeContentForDisplay(record.content)
    message += `${index + 1}. 内容: ${sanitizedContent}\n`
    if (showOriginalTime) {
      const originalTime = new Date(record.timestamp).toLocaleString('zh-CN')
      message += `   发送于: ${originalTime}\n`
    }
    message += `   撤回于: ${recallTime}\n\n`
  })

  return message.trim()
}

export function formatStatusMessage(status: any): string {
  const { globalEnabled, groupSpecificEnabled, effectiveConfig, statistics } = status
  const groupStatusText = groupSpecificEnabled === undefined
    ? `未单独设置 (跟随全局)`
    : `已单独设置为: ${formatBool(groupSpecificEnabled)}`

  return [
    `防撤回功能状态`,
    `全局默认: ${formatBool(globalEnabled)}`,
    `本群设置: ${groupStatusText}`,
    `---`,
    `当前生效状态: ${formatBool(effectiveConfig?.enabled || false)}`,
    `生效配置:`,
    `  - 保存天数: ${effectiveConfig?.retentionDays || 'N/A'} 天`,
    `  - 每用户最大记录: ${effectiveConfig?.maxRecordsPerUser || 'N/A'} 条`,
    `---`,
    `统计信息:`,
    `  - 总记录数: ${statistics.totalRecords}`,
    `  - 涉及用户数: ${statistics.totalUsers}`,
    `  - 涉及群组数: ${statistics.totalGuilds}`,
  ].join('\n')
}

function splitRecallArgs(input: string): string[] {
  if (!input.includes('<at')) {
    return input.split(/\s+/).filter(arg => arg)
  }

  const atMatch = input.match(/<at[^>]+>/)
  if (!atMatch) return input.split(/\s+/).filter(arg => arg)

  const atPart = atMatch[0]
  const restPart = input.replace(atPart, '').trim()
  return [atPart, ...restPart.split(/\s+/).filter(arg => arg)]
}

function parseTargetUserId(targetUser: string): string | null {
  if (targetUser.startsWith('<at')) {
    const match = targetUser.match(/id="([^"]+)"/)
    return match ? match[1] : null
  }
  return parseUserId(targetUser)
}

function parseUserId(input: string): string | null {
  if (!input) return null
  const cleaned = input.replace(/^@/, '').trim()
  if (/^\d+$/.test(cleaned)) return cleaned
  return null
}

function sanitizeContentForDisplay(content: string): string {
  if (!content) return '[空消息]'

  try {
    const elements = Element.parse(content)
    return elements.map(formatElement).join('').trim()
  } catch {
    return content.replace(/<[^>]+>/g, '').trim() || '[消息内容解析失败]'
  }
}

function formatElement(element: Element): string {
  switch (element.type) {
    case 'text':
      return element.attrs.content
    case 'face':
      return `[表情:${element.attrs.name || element.attrs.id}]`
    case 'img':
      return '[图片]'
    case 'at':
      return `[@${element.attrs.name || element.attrs.id}]`
    case 'video':
      return '[视频]'
    case 'audio':
      return '[语音]'
    case 'file':
      return '[文件]'
    default:
      return `[${element.type}]`
  }
}

function applyEnabledOption(options: any, updates: any, messages: string[]): void {
  if (options.enabled === undefined) return

  const enabledStr = options.enabled.toString().toLowerCase()
  if (['true', '1', 'yes', 'y', 'on'].includes(enabledStr)) {
    updates.enabled = true
    messages.push('已启用防撤回')
  } else if (['false', '0', 'no', 'n', 'off'].includes(enabledStr)) {
    updates.enabled = false
    messages.push('已禁用防撤回')
  }
}

function applyDaysOption(options: any, updates: any, messages: string[]): void {
  if (options.days === undefined) return

  if (options.days >= MIN_CONFIG_VALUE && options.days <= MAX_RETENTION_DAYS) {
    updates.retentionDays = options.days
    messages.push(`保留天数设为 ${options.days} 天`)
  } else {
    messages.push('保留天数无效 (需 1-365)')
  }
}

function applyMaxOption(options: any, updates: any, messages: string[]): void {
  if (options.max === undefined) return

  if (options.max >= MIN_CONFIG_VALUE && options.max <= MAX_RECORDS_PER_USER) {
    updates.maxRecordsPerUser = options.max
    messages.push(`最大记录数设为 ${options.max} 条`)
  } else {
    messages.push('最大记录数无效 (需 1-1000)')
  }
}

function formatBool(value: boolean): string {
  return value ? '已启用' : '已禁用'
}
