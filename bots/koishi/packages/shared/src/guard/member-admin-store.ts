import type { Context } from 'koishi'

import { GUARD_MEMBER_TABLE } from '../types/index'
import {
  activeGuardMemberQuery,
  type GuardMemberVersionRef,
} from './member-store-query'

export interface GuardMemberAdminRecord extends GuardMemberVersionRef {
  readonly guildId: string
  readonly channelId: string
  readonly memberId: string
  readonly deadlineAt: Date
  readonly releasedAt: Date | null
  readonly kickedAt: Date | null
}

export interface ListActiveGuardMembersByGuildInput {
  readonly guildId: string
  readonly memberIds?: readonly string[]
}

export interface MarkActiveGuardMemberMutedInput {
  readonly record: GuardMemberVersionRef
  readonly mutedAt: Date
}

export class GuardMemberAdminStore {
  constructor(private readonly ctx: Pick<Context, 'database'>) {}

  async listActiveByGuild(input: ListActiveGuardMembersByGuildInput) {
    const guildId = input.guildId.trim()
    if (!guildId) return []

    const records = await this.ctx.database.get(GUARD_MEMBER_TABLE, {
      guildId,
      releasedAt: null,
      kickedAt: null,
    }) as GuardMemberAdminRecord[]
    const memberIds = input.memberIds?.length ? new Set(input.memberIds) : undefined
    return records
      .filter((record) => !memberIds || memberIds.has(record.memberId))
      .sort((left, right) => left.deadlineAt.getTime() - right.deadlineAt.getTime())
  }

  async tryMarkActiveMuted(input: MarkActiveGuardMemberMutedInput) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberQuery(input.record), {
      mutedAt: input.mutedAt,
      updatedAt: input.mutedAt,
    })
    return result.matched === 1
  }
}
