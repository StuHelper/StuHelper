import type { DashboardPageData } from '../page-types'

import { formatTimestamp } from './formatters'

type DashboardTargetView = 'review' | 'identity' | 'config'

export interface DashboardTarget {
  view: DashboardTargetView
  workspace: string
  guildId: string | null
  memberId: string | null
  itemId: string | null
  keyword: string
}

export interface DashboardTodoRow {
  id: string
  title: string
  meta: string
  target: DashboardTarget
}

export interface DashboardActivityRow {
  id: string
  title: string
  meta: string
  sortTime: number
}

export const DASHBOARD_SHORTCUTS: ReadonlyArray<{
  label: string
  description: string
  target: DashboardTarget
}> = [
  {
    label: '查看全部复核',
    description: '进入处置中心查看待复核动作。',
    target: { view: 'review', workspace: 'review', guildId: null, memberId: null, itemId: null, keyword: '' },
  },
  {
    label: '处理待认证成员',
    description: '进入身份认证查看受限成员和最近释放记录。',
    target: { view: 'identity', workspace: 'members', guildId: null, memberId: null, itemId: null, keyword: '' },
  },
  {
    label: '检查模板与群绑定',
    description: '进入配置治理的模板和绑定工作区。',
    target: { view: 'config', workspace: 'bindings', guildId: null, memberId: null, itemId: null, keyword: '' },
  },
  {
    label: '检索事件与举报',
    description: '进入处置中心查看举报和上下文事件。',
    target: { view: 'review', workspace: 'report', guildId: null, memberId: null, itemId: null, keyword: '' },
  },
] as const

export function buildDashboardModel(data: DashboardPageData) {
  return {
    todoRows: buildDashboardTodoRows(data),
    shortcuts: DASHBOARD_SHORTCUTS,
    activityRows: buildDashboardActivityRows(data),
  }
}

function buildDashboardTodoRows(data: DashboardPageData): DashboardTodoRow[] {
  const reviewRows = data.pendingReviews.slice(0, 4).map((item) => ({
    id: item.id,
    title: `复核 · ${item.memberId}`,
    meta: `${item.reason} · ${formatTimestamp(item.createdAt)}`,
    target: {
      view: 'review' as const,
      workspace: 'review',
      guildId: item.guildId,
      memberId: item.memberId,
      itemId: item.id,
      keyword: '',
    },
  }))

  const memberRows = data.pendingMembers.slice(0, 4).map((item) => ({
    id: item.id,
    title: `认证 · ${item.memberName || item.memberId}`,
    meta: `${item.guildId} · 截止 ${formatTimestamp(item.deadlineAt)}`,
    target: {
      view: 'identity' as const,
      workspace: 'members',
      guildId: item.guildId,
      memberId: item.memberId,
      itemId: item.id,
      keyword: '',
    },
  }))

  return [...reviewRows, ...memberRows]
}

function buildDashboardActivityRows(data: DashboardPageData): DashboardActivityRow[] {
  const events = data.recentEvents.map((item) => ({
    id: item.id,
    title: item.summary,
    meta: `事件 · ${item.guildId} · ${formatTimestamp(item.createdAt)}`,
    sortTime: Date.parse(item.createdAt),
  }))
  const reports = data.recentReports.map((item) => ({
    id: item.id,
    title: item.aiSummary || item.reason,
    meta: `举报 · ${item.reporterMemberId} → ${item.targetMemberId} · ${formatTimestamp(item.createdAt)}`,
    sortTime: Date.parse(item.createdAt),
  }))

  return [...events, ...reports]
    .sort((left, right) => right.sortTime - left.sortTime)
    .slice(0, 8)
}
