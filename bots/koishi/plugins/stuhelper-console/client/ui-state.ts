import { describeAction, describeReviewAction, describeReviewStatus, describeVerificationState, formatTimestamp } from './formatters'

export type ConsoleInspectorKind =
  | 'member'
  | 'review'
  | 'event'
  | 'report'
  | 'template'
  | 'binding'
  | 'rule'

export interface DrawerItem {
  label: string
  value: string
  mono?: boolean
}

export interface DrawerSection {
  key: string
  title: string
  items: DrawerItem[]
}

interface DrawerField {
  label: string
  field: string
  mono?: boolean
  format?: (value: unknown) => string
}

const DEFAULT_NOTICE_MESSAGE = '操作已提交并刷新。'

export function resolveNoticeMessage(
  result: unknown,
  fallback = DEFAULT_NOTICE_MESSAGE,
) {
  return typeof result === 'string' && result.trim() ? result : fallback
}

export function buildInspectorSections(
  kind: ConsoleInspectorKind | null,
  payload: unknown,
): DrawerSection[] {
  if (!kind || !payload || typeof payload !== 'object') return []

  const record = payload as Record<string, unknown>

  switch (kind) {
    case 'member':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '成员', field: 'memberName' },
          { label: '成员 ID', field: 'memberId', mono: true },
        ]],
        ['context', '上下文记录', [
          { label: '群号', field: 'guildId', mono: true },
          { label: '认证状态', field: 'verificationState', format: formatVerificationState },
          { label: '截止时间', field: 'deadlineAt', mono: true, format: formatValueTimestamp },
        ]],
        ['risk', '风险与说明', [
          { label: '最后错误', field: 'lastError' },
        ]],
      ])
    case 'review':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '成员', field: 'memberId', mono: true },
          { label: '动作', field: 'actionType', format: formatReviewAction },
          { label: '状态', field: 'status', format: formatReviewStatus },
        ]],
        ['context', '上下文记录', [
          { label: '提交时间', field: 'createdAt', mono: true, format: formatValueTimestamp },
          { label: '处理备注', field: 'resolutionNote' },
        ]],
        ['risk', '风险与说明', [
          { label: '触发原因', field: 'reason' },
        ]],
      ])
    case 'event':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '类型', field: 'type' },
          { label: '级别', field: 'level' },
        ]],
        ['context', '上下文记录', [
          { label: '成员', field: 'memberId', mono: true },
          { label: '群号', field: 'guildId', mono: true },
          { label: '发生时间', field: 'createdAt', mono: true, format: formatValueTimestamp },
        ]],
        ['risk', '风险与说明', [
          { label: '摘要', field: 'summary' },
          { label: '原始载荷', field: 'payload' },
        ]],
      ])
    case 'report':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '举报人', field: 'reporterMemberId', mono: true },
          { label: '目标成员', field: 'targetMemberId', mono: true },
        ]],
        ['context', '上下文记录', [
          { label: '群号', field: 'guildId', mono: true },
          { label: '频道', field: 'channelId', mono: true },
          { label: '平台', field: 'platform', mono: true },
          { label: 'AI 状态', field: 'aiStatus' },
          { label: 'AI 等级', field: 'aiSeverity' },
          { label: '提交时间', field: 'createdAt', mono: true, format: formatValueTimestamp },
        ]],
        ['risk', '风险与说明', [
          { label: 'AI 摘要', field: 'aiSummary' },
          { label: '举报原因', field: 'reason' },
        ]],
      ])
    case 'template':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '模板名称', field: 'name' },
          { label: '模板 ID', field: 'id', mono: true },
        ]],
        ['context', '上下文记录', [
          { label: '禁言秒数', field: 'muteDurationSeconds', mono: true },
          { label: '踢出分钟数', field: 'kickAfterMinutes', mono: true },
          { label: '启用状态', field: 'enabled' },
        ]],
        ['risk', '风险与说明', [
          { label: '提醒文案', field: 'reminderTemplate' },
          { label: '白名单成员', field: 'exemptUsers' },
        ]],
      ])
    case 'binding':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '平台', field: 'platform' },
          { label: '群号', field: 'guildId', mono: true },
        ]],
        ['context', '上下文记录', [
          { label: '模板 ID', field: 'templateId', mono: true },
          { label: '启用状态', field: 'enabled' },
        ]],
        ['risk', '风险与说明', [
          { label: '备注', field: 'note' },
        ]],
      ])
    case 'rule':
      return createSections(record, [
        ['basic', '基本信息', [
          { label: '规则 ID', field: 'id', mono: true },
          { label: '群号', field: 'guildId', mono: true },
        ]],
        ['context', '上下文记录', [
          { label: '匹配模式', field: 'matchMode' },
          { label: '动作', field: 'action', format: formatRuleAction },
          { label: '启用状态', field: 'enabled' },
        ]],
        ['risk', '风险与说明', [
          { label: '表达式', field: 'pattern' },
          { label: '备注', field: 'note' },
          { label: '禁言秒数', field: 'muteSeconds', mono: true },
        ]],
      ])
    default:
      return []
  }
}

function createSections(
  record: Record<string, unknown>,
  groups: ReadonlyArray<
    readonly [
      key: string,
      title: string,
      fields: ReadonlyArray<DrawerField>,
    ]
  >,
) {
  return groups
    .map(([key, title, fields]) => ({
      key,
      title,
      items: fields
        .map((field) => createItem(field, record[field.field]))
        .filter((item): item is DrawerItem => item !== null),
    }))
    .filter((section) => section.items.length > 0)
}

function createItem(field: DrawerField, value: unknown) {
  const normalized = normalizeValue(field.format ? field.format(value) : value)
  if (!normalized) return null
  return {
    label: field.label,
    value: normalized,
    mono: Boolean(field.mono),
  }
}

function normalizeValue(value: unknown) {
  if (value === null || value === undefined || value === '') return ''
  if (Array.isArray(value)) return value.map((item) => String(item)).filter(Boolean).join(', ')
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function formatValueTimestamp(value: unknown) {
  return formatTimestamp(typeof value === 'string' ? value : '')
}

function formatReviewAction(value: unknown) {
  return describeReviewAction(String(value) as Parameters<typeof describeReviewAction>[0])
}

function formatReviewStatus(value: unknown) {
  return describeReviewStatus(String(value) as Parameters<typeof describeReviewStatus>[0])
}

function formatVerificationState(value: unknown) {
  return describeVerificationState(String(value) as Parameters<typeof describeVerificationState>[0])
}

function formatRuleAction(value: unknown) {
  return describeAction(String(value) as Parameters<typeof describeAction>[0]).label
}
