import type { ModerationReportRecord, ReviewStatus } from './types'

export interface IncrementWarningInput {
  readonly guildId: string
  readonly memberId: string
  readonly reason: string
  readonly now: Date
}

export interface ResolveReviewInput {
  readonly id: string
  readonly status: ReviewStatus
  readonly operatorMemberId: string
  readonly resolutionNote: string | null
}

export interface UpdateReportAIResultInput {
  readonly id: string
  readonly aiStatus: ModerationReportRecord['aiStatus']
  readonly aiSeverity: ModerationReportRecord['aiSeverity']
  readonly aiSummary: string | null
}
