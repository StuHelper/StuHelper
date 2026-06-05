import type { Element } from 'koishi'

export interface CommandLogEntry {
  timestamp: string | number
  guildId: string
  userId: string
  command: string
  target: string
  details: string
}

export interface CommandLogData {
  logs: unknown[]
  [key: string]: unknown
}

export interface WarnRecord {
  [userId: string]: {
    count: number
    timestamp: number
  }
}

export interface MuteRecord {
  startTime: number
  duration: number
  remainingTime?: number
  leftGroup?: boolean
  notified?: boolean
}

export interface BanMeRecord {
  count: number
  lastResetTime: number
  pity: number
  guaranteed: boolean
}

export interface LockedName {
  userId: string
  name: string
}

export interface LogRecord {
  time: string
  command: string
  user: string
  group: string
  target: string
  result: string
}

export interface LogSubscription {
  type: 'group' | 'private'
  id: string
}

export interface Subscription {
  type: 'group' | 'private'
  id: string
  features: {
    log?: boolean
    memberChange?: boolean
    muteExpire?: boolean
    blacklist?: boolean
    warning?: boolean
    antiRecall?: boolean
  }
}

export interface RepeatRecord {
  content: string
  count: number
  firstMessageId: string
  messages: Array<{
    id: string
    userId: string
    timestamp: number
  }>
}

export interface RecalledMessage {
  id: string
  messageId: string
  userId: string
  username: string
  guildId: string
  channelId?: string
  content: string
  timestamp: number
  recallTime: number
  elements?: Element[]
}

export interface RecallRecord {
  [guildId: string]: {
    [userId: string]: RecalledMessage[]
  }
}

export interface LeaveRecord {
  expireTime: number
}

export interface RegisteredCommand {
  name: string
  desc: string
  args?: string
  usage?: string
  examples?: string[]
  module: string
  moduleDesc: string
  permId?: string
  skipAuth?: boolean
}
