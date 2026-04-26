/**
 * 违规等级枚举
 */
export enum ViolationLevel {
  NONE = 0,
  LOW = 1,
  MEDIUM = 2,
  HIGH = 3,
  CRITICAL = 4,
}

export interface ViolationInfo {
  level: ViolationLevel
  reason: string
  action: ViolationAction[]
  reporterPenalty?: ReporterPenalty
}

export interface ViolationAction {
  type: 'ban' | 'warn' | 'kick' | 'kick_blacklist'
  time?: number
  count?: number
}

export interface ReporterPenalty {
  shouldLimit: boolean
  duration?: number
  reason?: string
}

export interface ReportBanRecord {
  userId: string
  guildId: string
  timestamp: number
  expireTime: number
}

export interface MessageRecord {
  userId: string
  content: string
  timestamp: number
}

export interface ReportedMessageRecord {
  messageId: string
  timestamp: number
  result: string
}
