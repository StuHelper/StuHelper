import type {
  StuhelperConsoleData,
  StuhelperConsoleEvent,
  StuhelperConsoleGuardMember,
  StuhelperConsoleReport,
  StuhelperConsoleReview,
} from '../../src/console-types'
import {
  describeReviewAction,
  describeReviewStatus,
  describeVerificationState,
  formatTimestamp,
} from '../formatters'
import type { ConsoleSearchState } from '../navigation'
import { buildRecentChanges } from './changes'
import { buildStatusBand, buildSystemStatus } from './status'

const REVIEW_TODO_LIMIT = 4
const IDENTITY_TODO_LIMIT = 4
const RECENT_ACTIVITY_LIMIT = 6
const RECENT_EVENT_LIMIT = 6
const REVIEW_QUEUE_ID = 'review'
const MEMBER_QUEUE_ID = 'member'
const POLICY_BINDINGS_QUEUE_ID = 'guard-bindings'

export type DashboardTarget = Pick<ConsoleSearchState, 'section' | 'queue' | 'id' | 'source'>

export interface DashboardStatusItem {
  label: string
  value: string
  note: string
  tone?: 'neutral' | 'warning' | 'danger' | 'success' | 'primary' | 'info'
}

export interface DashboardMetric {
  label: string
  value: number
  note: string
  tone?: 'neutral' | 'warning' | 'danger' | 'success' | 'primary'
}

export interface DashboardShortcut {
  label: string
  description: string
  target: DashboardTarget
}

export interface DashboardTodoRow {
  id: string
  kind: 'review' | 'identity'
  title: string
  meta: string
  status: string
  target: DashboardTarget
}

export interface DashboardActivityRow {
  id: string
  kind: 'event' | 'report'
  title: string
  meta: string
  time: string
}

export interface DashboardChangeRow {
  id: string
  title: string
  meta: string
  kindLabel: string
  time: string
}

export interface DashboardPolicySummary {
  label: string
  value: number
}

export interface DashboardModel {
  statusBand: DashboardStatusItem[]
  metrics: DashboardMetric[]
  todoRows: DashboardTodoRow[]
  systemStatus: DashboardStatusItem[]
  shortcuts: DashboardShortcut[]
  recentEvents: DashboardActivityRow[]
  policySummary: DashboardPolicySummary[]
  recentChanges: DashboardChangeRow[]
  recentActivity: DashboardActivityRow[]
}

interface SortableActivityRow extends DashboardActivityRow {
  sortTime: number
}

export function buildDashboardModel(data: StuhelperConsoleData | undefined): DashboardModel {
  const pendingReviews = data?.pendingReviews ?? []
  const pendingMembers = data?.pendingMembers ?? []

  return {
    statusBand: buildStatusBand(data),
    metrics: buildMetrics(data),
    todoRows: [
      ...buildReviewTodoRows(pendingReviews),
      ...buildIdentityTodoRows(pendingMembers),
    ],
    systemStatus: buildSystemStatus(data),
    shortcuts: buildShortcuts(),
    recentEvents: buildRecentEvents(data),
    policySummary: buildPolicySummary(data),
    recentChanges: buildRecentChanges(data),
    recentActivity: buildRecentActivity(data),
  }
}

function buildMetrics(data: StuhelperConsoleData | undefined): DashboardMetric[] {
  const reviewCount = data?.pendingReviews.length ?? 0
  const memberCount = data?.pendingMembers.length ?? 0
  const reportCount = data?.recentReports.length ?? 0
  const policyCount = countPolicyItems(data)

  return [
    createMetric('待复核', reviewCount, reviewCount > 0 ? '进入处置中心处理待复核动作' : '当前没有待复核动作'),
    createMetric('身份认证', memberCount, memberCount > 0 ? '进入身份认证分区处理待认证成员' : '当前没有待认证成员'),
    createMetric('最近举报', reportCount, reportCount > 0 ? `最近同步 ${reportCount} 条举报摘要` : '当前没有最近举报', 'primary'),
    createMetric('策略项', policyCount, policyCount > 0 ? `已配置 ${policyCount} 项策略` : '当前没有策略项', 'primary'),
  ]
}

function createMetric(
  label: string,
  value: number,
  note: string,
  tone?: DashboardMetric['tone'],
): DashboardMetric {
  if (tone) return { label, value, note, tone }
  if (value === 0) return { label, value, note, tone: 'success' }
  return { label, value, note, tone: 'warning' }
}

function buildShortcuts(): DashboardShortcut[] {
  return [
    createShortcut('查看全部复核', '进入处置中心查看待复核动作。', {
      section: 'enforcement',
      queue: REVIEW_QUEUE_ID,
      id: '',
      source: 'dashboard',
    }),
    createShortcut('处理待认证成员', '进入身份认证分区处理待认证成员。', {
      section: 'identity',
      queue: MEMBER_QUEUE_ID,
      id: '',
      source: 'dashboard',
    }),
    createShortcut('检查模板与群绑定', '进入策略中心检查模板、绑定和命令权限。', {
      section: 'policy',
      queue: POLICY_BINDINGS_QUEUE_ID,
      id: '',
      source: 'dashboard',
    }),
    createShortcut('检索事件与举报', '进入审计检索查看最近事件和举报摘要。', {
      section: 'audit',
      queue: null,
      id: '',
      source: 'dashboard',
    }),
  ]
}

function createShortcut(
  label: string,
  description: string,
  target: DashboardTarget,
): DashboardShortcut {
  return { label, description, target }
}

function buildReviewTodoRows(items: readonly StuhelperConsoleReview[]): DashboardTodoRow[] {
  return items.slice(0, REVIEW_TODO_LIMIT).map((item) => ({
    id: item.id,
    kind: 'review',
    title: item.reason || `${describeReviewAction(item.actionType)}复核`,
    meta: `${describeReviewAction(item.actionType)} · ${item.memberId} · ${formatTimestamp(item.createdAt)}`,
    status: describeReviewStatus(item.status),
    target: {
      section: 'enforcement',
      queue: REVIEW_QUEUE_ID,
      id: item.id,
      source: 'dashboard',
    },
  }))
}

function buildIdentityTodoRows(items: readonly StuhelperConsoleGuardMember[]): DashboardTodoRow[] {
  return [...items]
    .sort((left, right) => compareTime(left.deadlineAt, right.deadlineAt))
    .slice(0, IDENTITY_TODO_LIMIT)
    .map((item) => ({
      id: item.id,
      kind: 'identity',
      title: item.memberName || item.memberId,
      meta: `${item.memberId} · ${item.guildId} · 截止 ${formatTimestamp(item.deadlineAt)}`,
      status: describeVerificationState(item.verificationState),
      target: {
        section: 'identity',
        queue: MEMBER_QUEUE_ID,
        id: item.id,
        source: 'dashboard',
      },
    }))
}

function buildRecentEvents(data: StuhelperConsoleData | undefined): DashboardActivityRow[] {
  return [...(data?.recentEvents ?? [])]
    .sort((left, right) => compareTime(right.createdAt, left.createdAt))
    .slice(0, RECENT_EVENT_LIMIT)
    .map(toEventActivityRow)
}

function buildPolicySummary(data: StuhelperConsoleData | undefined): DashboardPolicySummary[] {
  return [
    { label: '关键词规则', value: data?.keywordRules.length ?? 0 },
    { label: '命令权限', value: data?.commandPolicies.length ?? 0 },
    { label: '成员角色', value: data?.memberRoles.length ?? 0 },
    { label: '守卫模板', value: data?.guardTemplates.length ?? 0 },
    { label: '群绑定', value: data?.guardBindings.length ?? 0 },
  ]
}

function buildRecentActivity(data: StuhelperConsoleData | undefined): DashboardActivityRow[] {
  const rows = [
    ...(data?.recentEvents ?? []).map(toEventActivityRow),
    ...(data?.recentReports ?? []).map(toReportActivityRow),
  ]

  return rows
    .sort((left, right) => right.sortTime - left.sortTime)
    .slice(0, RECENT_ACTIVITY_LIMIT)
    .map(({ sortTime: _sortTime, ...row }) => row)
}

function toEventActivityRow(item: StuhelperConsoleEvent): SortableActivityRow {
  return {
    id: item.id,
    kind: 'event',
    title: item.summary || item.type || '未命名事件',
    meta: `事件 · ${item.memberId || '—'} · ${item.guildId || '系统'}`,
    time: formatTimestamp(item.createdAt),
    sortTime: toTime(item.createdAt),
  }
}

function toReportActivityRow(item: StuhelperConsoleReport): SortableActivityRow {
  return {
    id: item.id,
    kind: 'report',
    title: item.aiSummary || item.reason || '未命名举报',
    meta: `举报 · ${item.reporterMemberId} → ${item.targetMemberId}`,
    time: formatTimestamp(item.createdAt),
    sortTime: toTime(item.createdAt),
  }
}

function countPolicyItems(data: StuhelperConsoleData | undefined) {
  return (
    (data?.keywordRules.length ?? 0) +
    (data?.commandPolicies.length ?? 0) +
    (data?.memberRoles.length ?? 0) +
    (data?.guardTemplates.length ?? 0) +
    (data?.guardBindings.length ?? 0)
  )
}

function compareTime(left: string, right: string) {
  return toTime(left) - toTime(right)
}

function toTime(value?: string | null) {
  if (!value) return 0
  const time = new Date(value).getTime()
  return Number.isNaN(time) ? 0 : time
}
