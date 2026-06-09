import type { Session } from 'koishi'

import type {
  AdmissionJoinRequestDecisionAction,
  PlatformClient,
} from '@stuhelper/koishi-shared'

export interface JoinRequestReviewRecordInput {
  readonly platform: string
  readonly guildId: string
  readonly memberId: string
  readonly requestID: string
  readonly decision: AdmissionJoinRequestDecisionAction
  readonly success: boolean
  readonly error?: string
  readonly rawEvent: Record<string, unknown>
}

interface JoinRequestReviewStrategyInput {
  readonly session: Session
  readonly platform: string
  readonly guildId: string
  readonly memberId: string
  readonly platformClient: PlatformClient
  readonly recordJoinRequest: (input: JoinRequestReviewRecordInput) => Promise<void>
}

export async function applyJoinRequestReviewStrategy(input: JoinRequestReviewStrategyInput) {
  const requestID = requestIDOf(input.session)
  const rawEvent = rawRequestEvent(input.session)
  const decision = await input.platformClient.resolveJoinRequestDecision({
    platform: input.platform,
    guildID: input.guildId,
    qqID: input.memberId,
    requestID,
    rawEvent,
  })
  const approve = decision.decision === 'approve'
  try {
    await input.session.bot.handleGuildMemberRequest(
      requestID,
      approve,
      approve ? undefined : decision.reason,
    )
    await input.recordJoinRequest({
      platform: input.platform,
      guildId: input.guildId,
      memberId: input.memberId,
      requestID,
      decision: decision.decision,
      success: true,
      rawEvent,
    })
  } catch (error) {
    await input.recordJoinRequest({
      platform: input.platform,
      guildId: input.guildId,
      memberId: input.memberId,
      requestID,
      decision: decision.decision,
      success: false,
      error: errorMessage(error),
      rawEvent,
    })
    throw error
  }
}

function requestIDOf(session: Session) {
  if (!session.messageId) {
    throw new Error('admission join request event missing message id')
  }
  return session.messageId
}

function rawRequestEvent(session: Session) {
  return (session.event as { _data?: Record<string, unknown> } | undefined)?._data || {}
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}
