import { Context } from 'koishi'

import { GUARD_MEMBER_TABLE, type PlatformVerificationState } from '../types/index'

declare module 'koishi' {
  interface Tables {
    [GUARD_MEMBER_TABLE]: GuardMemberRecord
  }
}

export interface GuardMemberRecord {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  memberName: string
  verificationState: PlatformVerificationState
  joinedAt: Date
  deadlineAt: Date
  mutedAt: Date | null
  reminderSentAt: Date | null
  releasedAt: Date | null
  kickedAt: Date | null
  lastError: string | null
  createdAt: Date
  updatedAt: Date
}

export function registerGuardMemberModel(ctx: Context) {
  ctx.model.extend(GUARD_MEMBER_TABLE, {
    id: 'string',
    platform: 'string',
    botSelfId: 'string',
    guildId: 'string',
    channelId: 'string',
    memberId: 'string',
    memberName: 'string',
    verificationState: 'string',
    joinedAt: 'timestamp',
    deadlineAt: 'timestamp',
    mutedAt: 'timestamp',
    reminderSentAt: 'timestamp',
    releasedAt: 'timestamp',
    kickedAt: 'timestamp',
    lastError: 'text',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, {
    primary: 'id',
  })
}

export * from './policy'
export * from './policy-store'
