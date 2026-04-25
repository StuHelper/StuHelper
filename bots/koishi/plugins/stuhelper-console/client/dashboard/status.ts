import type { StuhelperConsoleData } from '../../src/console-types'
import { formatTimestamp } from '../formatters'
import type { DashboardStatusItem } from './model'

export function buildStatusBand(data: StuhelperConsoleData | undefined): DashboardStatusItem[] {
  const reviewCount = data?.pendingReviews.length ?? 0
  const memberCount = data?.pendingMembers.length ?? 0
  const backlogCount = reviewCount + memberCount
  const highRiskCount = data?.overview.highRiskEvents ?? 0
  const reportCount = data?.overview.openReports ?? 0

  return [
    {
      label: '最后同步',
      value: data?.generatedAt ? formatTimestamp(data.generatedAt) : '未同步',
      note: data?.generatedAt ? '控制台数据服务已刷新' : '等待数据服务推送',
      tone: data?.generatedAt ? 'info' : 'warning',
    },
    {
      label: '积压总量',
      value: `${backlogCount} 项`,
      note: `复核 ${reviewCount} · 认证 ${memberCount}`,
      tone: backlogCount > 0 ? 'warning' : 'success',
    },
    {
      label: '风险摘要',
      value: `${highRiskCount} 条高风险`,
      note: `${reportCount} 条举报待跟进`,
      tone: highRiskCount > 0 || reportCount > 0 ? 'danger' : 'success',
    },
  ]
}

export function buildSystemStatus(data: StuhelperConsoleData | undefined): DashboardStatusItem[] {
  const reviewCount = data?.pendingReviews.length ?? 0
  const memberCount = data?.pendingMembers.length ?? 0

  return [
    {
      label: '数据服务',
      value: data?.generatedAt ? '运行中' : '未就绪',
      note: data?.generatedAt ? `最后同步 ${formatTimestamp(data.generatedAt)}` : '尚未收到数据快照',
      tone: data?.generatedAt ? 'success' : 'warning',
    },
    {
      label: '处置队列',
      value: reviewCount > 0 ? `${reviewCount} 条待处理` : '队列清零',
      note: '人工复核与高风险动作',
      tone: reviewCount > 0 ? 'warning' : 'success',
    },
    {
      label: '身份认证',
      value: memberCount > 0 ? `${memberCount} 名待认证` : '暂无积压',
      note: '成员准入与认证处理',
      tone: memberCount > 0 ? 'warning' : 'success',
    },
  ]
}
