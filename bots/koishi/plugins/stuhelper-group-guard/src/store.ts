import { Context } from 'koishi'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'

import type { GuardMemberRecord } from './model'

export class GuardMemberStore {
  constructor(private readonly ctx: Context) {}

  async savePending(record: GuardMemberRecord) {
    const existing = await this.getByID(record.id)
    if (!existing) {
      await this.ctx.database.create(GUARD_MEMBER_TABLE, record)
      return
    }
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id: record.id }, {
      platform: record.platform,
      botSelfId: record.botSelfId,
      guildId: record.guildId,
      channelId: record.channelId,
      memberId: record.memberId,
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
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id }, {
      mutedAt: now,
      updatedAt: now,
    })
  }

  async markReminderSent(id: string, now: Date) {
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id }, {
      reminderSentAt: now,
      updatedAt: now,
    })
  }

  async markReleased(id: string, now: Date) {
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id }, {
      releasedAt: now,
      lastError: null,
      updatedAt: now,
    })
  }

  async markKicked(id: string, now: Date) {
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id }, {
      kickedAt: now,
      lastError: null,
      updatedAt: now,
    })
  }

  async markLastError(id: string, message: string, now: Date) {
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id }, {
      lastError: message,
      updatedAt: now,
    })
  }

  async listActive() {
    return this.ctx.database.get(GUARD_MEMBER_TABLE, {
      releasedAt: null,
      kickedAt: null,
    }) as Promise<GuardMemberRecord[]>
  }

  async listPendingByGuild(guildId: string) {
    const records = await this.listActive()
    return records.filter((record) => record.guildId === guildId)
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
    await this.ctx.database.set(GUARD_MEMBER_TABLE, { id }, {
      ...input,
      lastError: null,
      updatedAt: new Date(),
    })
  }

  private async getByID(id: string) {
    const [record] = await this.ctx.database.get(GUARD_MEMBER_TABLE, { id })
    return record as GuardMemberRecord | undefined
  }
}
