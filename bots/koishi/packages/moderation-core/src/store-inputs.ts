import type { ModerationReportRecord, ReviewQueueRecord, ReviewStatus } from './types'

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

export interface UpdatePendingReviewInput {
  readonly review: Pick<ReviewQueueRecord, 'id' | 'updatedAt'>
  readonly status: ReviewStatus
  readonly operatorMemberId: string | null
  readonly resolutionNote: string | null
  readonly updatedAt: Date
}

export interface ClaimPendingReviewInput {
  readonly review: Pick<ReviewQueueRecord, 'id' | 'updatedAt'>
  readonly operatorMemberId: string
  readonly resolutionNote: string | null
  readonly claimedAt: Date
}

export interface FinalizeClaimedReviewInput {
  readonly reviewId: string
  readonly operatorMemberId: string
  readonly resolutionNote: string | null
  readonly claimedAt: Date
  readonly executedAt: Date
}

export interface RollbackClaimedReviewInput {
  readonly reviewId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
}

export interface UpdateReportAIResultInput {
  readonly id: string
  readonly aiStatus: ModerationReportRecord['aiStatus']
  readonly aiSeverity: ModerationReportRecord['aiSeverity']
  readonly aiSummary: string | null
}
