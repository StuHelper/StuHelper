import type { Context, Session } from 'koishi'
import { Logger } from 'koishi'
import { testSafeKeywordRegex, type PlatformClient } from '@stuhelper/koishi-shared'

import type { DataManager } from '../data'
import type { Config, GroupConfig } from '../../types'

export const eventLogger = new Logger('stuhelperGroupCenter:event')
export const DEFAULT_LEVEL_LIMIT = 0
export const DEFAULT_LEAVE_COOLDOWN_DAYS = 0
export const MUTE_EXPIRE_CHECK_INTERVAL_MS = 60_000
export const DEFAULT_MEMBER_REQUEST_CONFIG: GroupConfig = Object.freeze({
  keywords: [],
  approvalKeywords: [],
  levelLimit: DEFAULT_LEVEL_LIMIT,
})
export const DEFAULT_LEAVE_CONFIG: GroupConfig = Object.freeze({
  leaveCooldown: DEFAULT_LEAVE_COOLDOWN_DAYS,
})

export interface EventRuntimeHost {
  readonly ctx: Context
  readonly data: DataManager
  readonly config: Config
  readonly admissionPlatform?: Pick<
    PlatformClient,
    'getAdmissionQQAccess' | 'getMemberBlacklistAccess' | 'recordJoinRequestEvent'
  >
}

export type EventSession = Session & {
  readonly guildId: string
  readonly userId: string
}

export interface GroupRequest {
  readonly session: EventSession
  readonly failureLog: string
}

export function requestCommentOf(session: EventSession): string {
  return session.content || ''
}

export function requestIdOf(session: EventSession): string {
  if (!session.messageId) {
    throw new Error('request event missing message id')
  }
  return session.messageId
}

export function botInternal(session: EventSession): any {
  return (session.bot as any).internal
}

export function groupConfigOf(
  host: EventRuntimeHost,
  guildId: string,
  fallback: GroupConfig,
): GroupConfig {
  return host.data.groupConfig.getAll()[guildId] || fallback
}

export function matchesAnyKeyword(comment: string, keywords: readonly string[]): boolean {
  return keywords.some(keyword => matchesKeyword(comment, keyword))
}

function matchesKeyword(comment: string, keyword: string): boolean {
  return testSafeKeywordRegex(keyword, comment)
}
