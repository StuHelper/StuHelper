import type { DataManager } from '../data'
import type { Subscription, WarnRecord } from '../../types'

const NUMERIC_ID_PATTERN = /^\d+$/
const USER_ID_PATTERN = /^[a-zA-Z0-9_-]+$/
const SHANGHAI_TIMESTAMP_FORMATTER = new Intl.DateTimeFormat('sv-SE', {
  timeZone: 'Asia/Shanghai',
  hour12: false,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

type LegacyWarnRecord = {
  groups: Record<string, WarnRecord[string]>
}

export interface WarmCacheTargets {
  guildIds: string[]
  userIds: string[]
  memberPairs: Array<{ guildId: string; userId: string }>
}

export function collectWarmCacheTargets(
  data: DataManager,
  subscriptions: Subscription[],
): WarmCacheTargets {
  const guildIds = new Set<string>()
  const userIds = new Set<string>()
  const memberPairs: WarmCacheTargets['memberPairs'] = []

  collectGuildConfigTargets(data, guildIds)
  collectWarnTargets(data, guildIds, userIds, memberPairs)
  collectRecallTargets(data, guildIds, userIds, memberPairs)
  collectSubscriptionTargets(subscriptions, guildIds, userIds)

  return {
    guildIds: Array.from(guildIds),
    userIds: Array.from(userIds),
    memberPairs,
  }
}

export function formatShanghaiTimestamp(date: Date): string {
  return SHANGHAI_TIMESTAMP_FORMATTER.format(date).replace(',', '')
}

export function toErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function collectGuildConfigTargets(data: DataManager, guildIds: Set<string>) {
  Object.keys(data.groupConfig.getAll()).forEach((guildId) => {
    if (NUMERIC_ID_PATTERN.test(guildId)) {
      guildIds.add(guildId)
    }
  })
}

function collectWarnTargets(
  data: DataManager,
  guildIds: Set<string>,
  userIds: Set<string>,
  memberPairs: WarmCacheTargets['memberPairs'],
) {
  Object.entries(data.warns.getAll()).forEach(([key, value]) => {
    if (!isRecord(value)) {
      return
    }

    if (isLegacyWarnRecord(value)) {
      userIds.add(key)
      Object.keys(value.groups).forEach((guildId) => {
        if (NUMERIC_ID_PATTERN.test(guildId)) {
          guildIds.add(guildId)
          memberPairs.push({ guildId, userId: key })
        }
      })
      return
    }

    const guildId = key
    if (NUMERIC_ID_PATTERN.test(guildId)) {
      guildIds.add(guildId)
    }

    Object.keys(value).forEach((userId) => {
      if (!USER_ID_PATTERN.test(userId)) {
        return
      }
      userIds.add(userId)
      memberPairs.push({ guildId, userId })
    })
  })
}

function collectRecallTargets(
  data: DataManager,
  guildIds: Set<string>,
  userIds: Set<string>,
  memberPairs: WarmCacheTargets['memberPairs'],
) {
  Object.entries(data.recallRecords.getAll()).forEach(([guildId, users]) => {
    if (!isRecord(users)) {
      return
    }
    if (NUMERIC_ID_PATTERN.test(guildId)) {
      guildIds.add(guildId)
    }
    Object.keys(users).forEach((userId) => {
      if (!USER_ID_PATTERN.test(userId)) {
        return
      }
      userIds.add(userId)
      memberPairs.push({ guildId, userId })
    })
  })
}

function collectSubscriptionTargets(
  subscriptions: Subscription[],
  guildIds: Set<string>,
  userIds: Set<string>,
) {
  subscriptions.forEach((sub) => {
    if (sub.type === 'group' && NUMERIC_ID_PATTERN.test(sub.id)) {
      guildIds.add(sub.id)
      return
    }
    if (sub.type === 'private') {
      userIds.add(sub.id)
    }
  })
}

function isLegacyWarnRecord(value: Record<string, unknown>): value is LegacyWarnRecord {
  return 'groups' in value && isRecord(value.groups)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
