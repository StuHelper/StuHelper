import type { Context } from 'koishi'

import { GUARD_MEMBER_TABLE } from '../types/index'
import {
  activeGuardMemberQuery,
  claimedGuardMemberQuery,
  type GuardMemberVersionRef,
} from './member-store-query'

export type { GuardMemberVersionRef } from './member-store-query'

export interface ClaimActiveGuardMemberInput {
  readonly record: GuardMemberVersionRef
  readonly claimedAt: Date
}

export interface ReleaseClaimedGuardMemberInput {
  readonly guardId: string
  readonly claimedAt: Date
  readonly releasedAt: Date
}

export interface KickClaimedGuardMemberInput {
  readonly guardId: string
  readonly claimedAt: Date
  readonly kickedAt: Date
}

export interface DeferActiveGuardMemberInput {
  readonly record: GuardMemberVersionRef
  readonly deadlineAt: Date
  readonly updatedAt: Date
}

export interface RollbackGuardMemberClaimInput {
  readonly guardId: string
  readonly claimedAt: Date
  readonly rolledBackAt: Date
  readonly error: unknown
}

export interface MarkActiveGuardMemberKickedInput {
  readonly botSelfId: string
  readonly guildId: string
  readonly memberId: string
  readonly kickedAt: Date
}

interface GuardMemberLookupRecord extends GuardMemberVersionRef {
  readonly releasedAt: Date | null
  readonly kickedAt: Date | null
}

export class GuardMemberWorkItemStore {
  constructor(private readonly ctx: Pick<Context, 'database'>) {}

  async tryClaimActive(input: ClaimActiveGuardMemberInput) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberQuery(input.record), {
      updatedAt: input.claimedAt,
    })
    return result.matched === 1
  }

  async tryReleaseClaimed(input: ReleaseClaimedGuardMemberInput) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, claimedGuardMemberQuery(input), {
      releasedAt: input.releasedAt,
      lastError: null,
      updatedAt: input.releasedAt,
    })
    return result.matched === 1
  }

  async tryKickClaimed(input: KickClaimedGuardMemberInput) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, claimedGuardMemberQuery(input), {
      kickedAt: input.kickedAt,
      lastError: null,
      updatedAt: input.kickedAt,
    })
    return result.matched === 1
  }

  async tryDeferActive(input: DeferActiveGuardMemberInput) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberQuery(input.record), {
      deadlineAt: input.deadlineAt,
      lastError: null,
      updatedAt: input.updatedAt,
    })
    return result.matched === 1
  }

  async rollbackClaim(input: RollbackGuardMemberClaimInput) {
    await this.ctx.database.set(GUARD_MEMBER_TABLE, claimedGuardMemberQuery(input), {
      lastError: input.error instanceof Error ? input.error.message : String(input.error),
      updatedAt: input.rolledBackAt,
    })
  }

  async tryMarkActiveMemberKicked(input: MarkActiveGuardMemberKickedInput) {
    const [record] = await this.ctx.database.get(GUARD_MEMBER_TABLE, {
      botSelfId: input.botSelfId,
      guildId: input.guildId,
      memberId: input.memberId,
      releasedAt: null,
      kickedAt: null,
    }) as GuardMemberLookupRecord[]
    if (!record) return false

    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberQuery(record), {
      kickedAt: input.kickedAt,
      lastError: null,
      updatedAt: input.kickedAt,
    })
    return result.matched === 1
  }
}
