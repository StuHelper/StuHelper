import type { Context } from 'koishi'

import { GUARD_MEMBER_TABLE } from '../types/index'
import type { GuardMemberRecord } from './member-model'
import { activeGuardMemberIDQuery } from './member-store-query'

export class GuardMemberStore {
  constructor(private readonly ctx: Pick<Context, 'database'>) {}

  async savePending(record: GuardMemberRecord) {
    const existing = await this.getByID(record.id)
    if (!existing) {
      await this.ctx.database.create(GUARD_MEMBER_TABLE, record)
      return
    }
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, {
      channelId: record.channelId,
      memberName: record.memberName,
      verificationState: record.verificationState,
      admissionSessionID: record.admissionSessionID,
      backendSyncPending: record.backendSyncPending,
      joinedAt: record.joinedAt,
      deadlineAt: record.deadlineAt,
      nextReminderAt: record.nextReminderAt,
      manualReviewDeadlineAt: record.manualReviewDeadlineAt,
      mutedAt: null,
      reminderSentAt: null,
      releasedAt: null,
      kickedAt: null,
      lastError: record.lastError ?? existing.lastError,
      updatedAt: record.updatedAt,
    })
  }

  async markMuted(id: string, now: Date) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberIDQuery(id), {
      mutedAt: now,
      updatedAt: now,
    })
    return result.matched === 1
  }

  async markReminderSent(id: string, now: Date) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberIDQuery(id), {
      reminderSentAt: now,
      updatedAt: now,
    })
    return result.matched === 1
  }

  async markReleased(id: string, now: Date) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberIDQuery(id), {
      releasedAt: now,
      lastError: null,
      updatedAt: now,
    })
    return result.matched === 1
  }

  async markKicked(id: string, now: Date) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberIDQuery(id), {
      kickedAt: now,
      lastError: null,
      updatedAt: now,
    })
    return result.matched === 1
  }

  async markLastError(id: string, message: string, now: Date) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberIDQuery(id), {
      lastError: message,
      updatedAt: now,
    })
    return result.matched === 1
  }

  async listActive() {
    return this.ctx.database.get(GUARD_MEMBER_TABLE, {
      releasedAt: null,
      kickedAt: null,
    }) as Promise<GuardMemberRecord[]>
  }

  async listPendingByGuild(guildId: string) {
    return this.ctx.database.get(GUARD_MEMBER_TABLE, {
      guildId,
      releasedAt: null,
      kickedAt: null,
    }) as Promise<GuardMemberRecord[]>
  }

  async findActiveByAdmissionSessionID(sessionID: string) {
    const [record] = await this.ctx.database.get(GUARD_MEMBER_TABLE, {
      admissionSessionID: sessionID,
      releasedAt: null,
      kickedAt: null,
    })
    return record as GuardMemberRecord | undefined
  }

  async getActiveByID(id: string) {
    const [record] = await this.ctx.database.get(GUARD_MEMBER_TABLE, {
      id,
      releasedAt: null,
      kickedAt: null,
    })
    return record as GuardMemberRecord | undefined
  }

  async findActiveBySubject(input: {
    readonly platform: string
    readonly botSelfId: string
    readonly guildId: string
    readonly memberId: string
  }) {
    const [record] = await this.ctx.database.get(GUARD_MEMBER_TABLE, {
      platform: input.platform,
      botSelfId: input.botSelfId,
      guildId: input.guildId,
      memberId: input.memberId,
      releasedAt: null,
      kickedAt: null,
    })
    return record as GuardMemberRecord | undefined
  }

  async listBackendSyncPending(platform: string | undefined, botSelfId: string) {
    const query: Record<string, unknown> = {
      botSelfId,
      backendSyncPending: true,
      releasedAt: null,
      kickedAt: null,
    }
    if (platform) {
      query.platform = platform
    }
    return this.ctx.database.get(GUARD_MEMBER_TABLE, query) as Promise<GuardMemberRecord[]>
  }

  async markBackendSynced(
    id: string,
    input: Pick<GuardMemberRecord,
      'admissionSessionID' | 'deadlineAt' | 'nextReminderAt' | 'manualReviewDeadlineAt'
    > & { backendSyncPending: false },
  ) {
    const result = await this.ctx.database.set(GUARD_MEMBER_TABLE, activeGuardMemberIDQuery(id), {
      ...input,
      lastError: null,
      updatedAt: new Date(),
    })
    return result.matched === 1
  }

  private async getByID(id: string) {
    const [record] = await this.ctx.database.get(GUARD_MEMBER_TABLE, { id })
    return record as GuardMemberRecord | undefined
  }
}
