import type {
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type { GuardMemberRecord } from '@stuhelper/koishi-shared'

import { serializeEvent } from './page-serializers'
import type {
  ReviewPageData,
  ReviewWorkItem,
  ReviewWorkItemAction,
} from './page-types'

export interface ReviewPageBuilderInput {
  generatedAt: string
  pendingReviews: ReviewQueueRecord[]
  pendingMembers: GuardMemberRecord[]
  reports: ModerationReportRecord[]
  events: ModerationEventRecord[]
}

export function buildReviewPageData(input: ReviewPageBuilderInput): ReviewPageData {
  const items = [
    ...input.pendingReviews.map((review) => toReviewItem(review)),
    ...input.pendingMembers.filter((item) => !item.releasedAt && !item.kickedAt).map((member) => toAdmissionItem(member)),
    ...input.reports.map((report) => toReportItem(report, input.events)),
  ].sort((left, right) => Date.parse(right.createdAt) - Date.parse(left.createdAt))

  return {
    generatedAt: input.generatedAt,
    items,
    events: [...input.events].sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime()).map(serializeEvent),
  }
}

export interface ReviewPageServiceDeps {
  loadPendingReviews: () => Promise<ReviewQueueRecord[]>
  loadPendingMembers: () => Promise<GuardMemberRecord[]>
  loadReports: () => Promise<ModerationReportRecord[]>
  loadEvents: () => Promise<ModerationEventRecord[]>
}

export class ReviewPageService {
  constructor(private readonly deps: ReviewPageServiceDeps) {}

  async getPageData() {
    const [pendingReviews, pendingMembers, reports, events] = await Promise.all([
      this.deps.loadPendingReviews(),
      this.deps.loadPendingMembers(),
      this.deps.loadReports(),
      this.deps.loadEvents(),
    ])

    return buildReviewPageData({
      generatedAt: new Date().toISOString(),
      pendingReviews,
      pendingMembers,
      reports,
      events,
    })
  }
}

function toReviewItem(review: ReviewQueueRecord): ReviewWorkItem {
  return {
    id: review.id,
    kind: 'review',
    guildId: review.guildId,
    memberId: review.memberId,
    subjectLabel: review.memberId,
    status: review.status,
    priority: review.actionType === 'kick_and_block' ? 'high' : 'medium',
    createdAt: review.createdAt.toISOString(),
    availableActions: ['execute', 'reject'],
    relatedEventIds: [],
    reason: review.reason,
    secondaryLabel: review.actionType,
  }
}

function toAdmissionItem(member: GuardMemberRecord): ReviewWorkItem {
  return {
    id: member.id,
    kind: 'admission',
    guildId: member.guildId,
    memberId: member.memberId,
    subjectLabel: member.memberName || member.memberId,
    status: member.verificationState,
    priority: 'medium',
    createdAt: member.createdAt.toISOString(),
    availableActions: ['approve', 'deny', 'defer'],
    relatedEventIds: [],
    reason: member.lastError || '等待认证完成',
    secondaryLabel: member.guildId,
  }
}

function toReportItem(report: ModerationReportRecord, events: readonly ModerationEventRecord[]): ReviewWorkItem {
  return {
    id: report.id,
    kind: 'report',
    guildId: report.guildId,
    memberId: report.targetMemberId,
    subjectLabel: report.targetMemberId,
    status: report.aiStatus,
    priority: toReportPriority(report.aiSeverity),
    createdAt: report.createdAt.toISOString(),
    availableActions: ['dismiss', 'escalate', 'create-review'],
    relatedEventIds: resolveRelatedEventIds(report, events),
    reason: report.reason,
    secondaryLabel: report.reporterMemberId,
  }
}

function resolveRelatedEventIds(report: ModerationReportRecord, events: readonly ModerationEventRecord[]) {
  return events
    .filter((event) => {
      const payloadReportId = typeof event.payload?.reportId === 'string'
        ? event.payload.reportId
        : typeof event.payload?.reportID === 'string'
          ? event.payload.reportID
          : null
      return payloadReportId === report.id
        || (event.guildId === report.guildId && event.memberId === report.targetMemberId)
    })
    .map((event) => event.id)
}

function toReportPriority(severity: ModerationReportRecord['aiSeverity']): ReviewWorkItem['priority'] {
  switch (severity) {
    case 'high':
      return 'critical'
    case 'medium':
      return 'high'
    case 'low':
      return 'medium'
    default:
      return 'low'
  }
}
