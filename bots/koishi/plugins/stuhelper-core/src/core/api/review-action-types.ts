import type { Context } from 'koishi'
import type {
  ClaimPendingReviewInput,
  FinalizeClaimedReviewInput,
  KickMemberInput,
  RollbackClaimedReviewInput,
  ReviewActionType,
  UpdatePendingReviewInput,
} from '@stuhelper/koishi-moderation-core'
import type { GuardMemberWorkItemStore } from '@stuhelper/koishi-shared'

import type { ConsoleGuildScope } from './console-guild-scope'

export const REVIEW_STATUS_PENDING = 'pending'
export const REVIEW_STATUS_APPROVED = 'approved'
export const REVIEW_STATUS_REJECTED = 'rejected'
export const REVIEW_STATUS_EXECUTED = 'executed'

export type ReviewActionInput = {
  kind: 'review'
  itemId: string
  action: 'execute' | 'reject'
  note?: string
}

export type AdmissionActionInput = {
  kind: 'admission'
  itemId: string
  action: 'approve' | 'deny' | 'defer'
  note?: string
}

export type ReportActionInput = {
  kind: 'report'
  itemId: string
  action: 'dismiss' | 'escalate' | 'create-review'
  note?: string
}

export type WorkItemActionInput =
  | ReviewActionInput
  | AdmissionActionInput
  | ReportActionInput

export interface WorkItemActionDeps {
  ctx: Context
  moderationStore: {
    appendEvent: (input: Record<string, unknown>) => Promise<unknown>
    createReview: (input: Record<string, unknown>) => Promise<{ id: string; actionType: ReviewActionType }>
    removeReport?: (id: string) => Promise<unknown>
    rollbackClaimedReview: (input: RollbackClaimedReviewInput) => Promise<unknown>
    transact?: <T>(callback: (store: WorkItemActionDeps['moderationStore']) => Promise<T>) => Promise<T>
    tryClaimPendingReview: (input: ClaimPendingReviewInput) => Promise<boolean>
    tryFinalizeClaimedReview: (input: FinalizeClaimedReviewInput) => Promise<boolean>
    tryUpdatePendingReview: (input: UpdatePendingReviewInput) => Promise<boolean>
    updateReportAIResult: (input: {
      readonly id: string
      readonly aiStatus: string
      readonly aiSeverity: string
      readonly aiSummary: string | null
    }) => Promise<unknown>
  }
  guardMemberStore: Pick<GuardMemberWorkItemStore,
    'rollbackClaim' |
    'tryClaimActive' |
    'tryDeferActive' |
    'tryKickClaimed' |
    'tryReleaseClaimed'
  >
  actions: {
    kickMember: (input: KickMemberInput) => Promise<unknown>
  }
  now?: () => Date
}

export interface WorkItemActionActor {
  memberId: string
  displayName: string | null
  guildScope: ConsoleGuildScope
}

export function getNow(deps: WorkItemActionDeps) {
  return deps.now ? deps.now() : new Date()
}

export function toErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}
