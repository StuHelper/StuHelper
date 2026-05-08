const CACHE_VERSION = '1.0.0'
const CACHE_EXPIRY_MS = 7 * 24 * 60 * 60 * 1000

export interface GuildCacheInfo {
  id: string
  name: string
  avatar?: string
  lastUpdate: number
}

export interface UserCacheInfo {
  id: string
  name: string
  avatar?: string
  lastUpdate: number
}

export interface MemberCacheInfo {
  guildId: string
  userId: string
  nick?: string
  name?: string
  avatar?: string
  lastUpdate: number
}

export interface CacheData {
  guilds: Record<string, GuildCacheInfo>
  users: Record<string, UserCacheInfo>
  members: Record<string, MemberCacheInfo>
  metadata: {
    lastFullRefresh: number
    version: string
  }
  [key: string]: unknown
}

export function createEmptyCacheData(): CacheData {
  return {
    guilds: {},
    users: {},
    members: {},
    metadata: {
      lastFullRefresh: 0,
      version: CACHE_VERSION,
    },
  }
}

export function isCacheFresh(value: { lastUpdate: number } | undefined, now = Date.now()) {
  return Boolean(value && now - value.lastUpdate < CACHE_EXPIRY_MS)
}

export function memberCacheKey(guildId: string, userId: string) {
  return `${guildId}:${userId}`
}

export function qqGuildAvatar(guildId: string) {
  return `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/`
}

export function qqUserAvatar(userId: string) {
  return `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
}

export function isQQPlatform(platform: string | undefined) {
  return platform === 'onebot' || platform === 'red' || platform === 'qq'
}
