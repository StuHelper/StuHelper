import type { Context, Session } from 'koishi'
import { Logger } from 'koishi'

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
}

export interface RequestData {
  flag: string
  sub_type: string
  comment?: string
}

export type EventSession = Session & {
  readonly guildId: string
  readonly userId: string
}

export interface GroupRequest {
  readonly session: EventSession
  readonly data: RequestData
  readonly failureLog: string
}

export function requestDataOf(session: EventSession): RequestData {
  return (session.event as { _data: RequestData })._data
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
  try {
    return new RegExp(keyword, 'i').test(comment)
  } catch {
    return comment.toLowerCase().includes(keyword.toLowerCase())
  }
}
