import type {
  StuhelperConsoleEvent,
  StuhelperConsoleReport,
} from '../../src/console-types'

const AUDIT_FILTER_KINDS = ['all', 'event', 'report'] as const

export type AuditFilterKind = (typeof AUDIT_FILTER_KINDS)[number]
export type AuditRowKind = Exclude<AuditFilterKind, 'all'>

export interface AuditRow {
  id: string
  kind: AuditRowKind
  createdAt: string
  summary: string
  memberId: string
  target: string
  level: string
  detail: string
}

export interface BuildAuditRowsOptions {
  query?: string
  kind?: AuditFilterKind | string | null
}

interface SearchableAuditRow extends AuditRow {
  sortTime: number
}

export function normalizeAuditFilterKind(value?: string | null): AuditFilterKind {
  return AUDIT_FILTER_KINDS.includes(value as AuditFilterKind) ? (value as AuditFilterKind) : 'all'
}

export function buildAuditRows(
  events: readonly StuhelperConsoleEvent[],
  reports: readonly StuhelperConsoleReport[],
  options: BuildAuditRowsOptions = {},
): AuditRow[] {
  const query = options.query?.trim().toLowerCase() ?? ''
  const kind = normalizeAuditFilterKind(options.kind)

  return [...events.map(toEventRow), ...reports.map(toReportRow)]
    .filter((row) => kind === 'all' || row.kind === kind)
    .filter((row) => matchesAuditQuery(row, query))
    .sort((left, right) => right.sortTime - left.sortTime)
    .map(({ sortTime: _sortTime, ...row }) => row)
}

function toEventRow(event: StuhelperConsoleEvent): SearchableAuditRow {
  return {
    id: event.id,
    kind: 'event',
    createdAt: event.createdAt,
    summary: event.summary || event.type || '未命名事件',
    memberId: event.memberId || '—',
    target: event.guildId || '系统',
    level: event.level || '—',
    detail: event.type || 'event',
    sortTime: toTime(event.createdAt),
  }
}

function toReportRow(report: StuhelperConsoleReport): SearchableAuditRow {
  return {
    id: report.id,
    kind: 'report',
    createdAt: report.createdAt,
    summary: report.aiSummary || report.reason || '未命名举报',
    memberId: report.reporterMemberId || '—',
    target: report.targetMemberId || '—',
    level: report.aiSeverity || '—',
    detail: report.aiStatus || 'report',
    sortTime: toTime(report.createdAt),
  }
}

function matchesAuditQuery(row: SearchableAuditRow, query: string) {
  if (!query) return true
  return [row.id, row.kind, row.summary, row.memberId, row.target, row.level, row.detail]
    .some((field) => field.toLowerCase().includes(query))
}

function toTime(value: string) {
  const time = Date.parse(value)
  return Number.isNaN(time) ? 0 : time
}
