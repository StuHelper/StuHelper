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
        ['basic', '基本信息', [['成员', 'memberName'], ['成员 ID', 'memberId', true]]],
        ['context', '上下文记录', [['群号', 'guildId', true], ['认证状态', 'verificationState'], ['截止时间', 'deadlineAt', true]]],
        ['risk', '风险与说明', [['最后错误', 'lastError']]],
      ])
    case 'review':
      return createSections(record, [
        ['basic', '基本信息', [['成员', 'memberId', true], ['动作', 'actionType'], ['状态', 'status']]],
        ['context', '上下文记录', [['提交时间', 'createdAt', true], ['处理备注', 'resolutionNote']]],
        ['risk', '风险与说明', [['触发原因', 'reason']]],
      ])
    case 'event':
      return createSections(record, [
        ['basic', '基本信息', [['类型', 'type'], ['级别', 'level']]],
        ['context', '上下文记录', [['成员', 'memberId', true], ['群号', 'guildId', true], ['发生时间', 'createdAt', true]]],
        ['risk', '风险与说明', [['摘要', 'summary'], ['原始载荷', 'payload']]],
      ])
    case 'report':
      return createSections(record, [
        ['basic', '基本信息', [['举报人', 'reporterMemberId', true], ['目标成员', 'targetMemberId', true]]],
        ['context', '上下文记录', [['AI 状态', 'aiStatus'], ['AI 等级', 'aiSeverity'], ['提交时间', 'createdAt', true]]],
        ['risk', '风险与说明', [['AI 摘要', 'aiSummary'], ['举报原因', 'reason']]],
      ])
    case 'template':
      return createSections(record, [
        ['basic', '基本信息', [['模板名称', 'name'], ['模板 ID', 'id', true]]],
        ['context', '上下文记录', [['禁言秒数', 'muteDurationSeconds', true], ['踢出分钟数', 'kickAfterMinutes', true], ['启用状态', 'enabled']]],
        ['risk', '风险与说明', [['提醒文案', 'reminderTemplate'], ['白名单成员', 'exemptUsers']]],
      ])
    case 'binding':
      return createSections(record, [
        ['basic', '基本信息', [['平台', 'platform'], ['群号', 'guildId', true]]],
        ['context', '上下文记录', [['模板 ID', 'templateId', true], ['启用状态', 'enabled']]],
        ['risk', '风险与说明', [['备注', 'note']]],
      ])
    case 'rule':
      return createSections(record, [
        ['basic', '基本信息', [['规则 ID', 'id', true], ['群号', 'guildId', true]]],
        ['context', '上下文记录', [['匹配模式', 'matchMode'], ['动作', 'action'], ['启用状态', 'enabled']]],
        ['risk', '风险与说明', [['表达式', 'pattern'], ['备注', 'note'], ['禁言秒数', 'muteSeconds', true]]],
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
      fields: ReadonlyArray<readonly [label: string, field: string, mono?: boolean]>,
    ]
  >,
) {
  return groups
    .map(([key, title, fields]) => ({
      key,
      title,
      items: fields
        .map(([label, field, mono]) => createItem(label, record[field], Boolean(mono)))
        .filter((item): item is DrawerItem => item !== null),
    }))
    .filter((section) => section.items.length > 0)
}

function createItem(label: string, value: unknown, mono: boolean) {
  const normalized = normalizeValue(value)
  if (!normalized) return null
  return {
    label,
    value: normalized,
    mono,
  }
}

function normalizeValue(value: unknown) {
  if (value === null || value === undefined || value === '') return ''
  if (Array.isArray(value)) return value.map((item) => String(item)).filter(Boolean).join(', ')
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
