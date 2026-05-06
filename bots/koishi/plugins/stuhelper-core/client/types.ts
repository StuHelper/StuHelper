/**
 * 客户端类型定义
 * 注意：这里直接定义类型，不从 src 导入（client 和 src 有独立的 tsconfig）
 */

export interface RoleMember {
 id: string
 name: string
 avatar: string
}

// 群组配置
export interface GroupConfig {
  guildName?: string
  guildAvatar?: string
  keywords?: string[]
  approvalKeywords?: string[]
  auto?: string
  reject?: string
  forbidden?: {
    autoDelete: boolean
    autoBan: boolean
    autoKick: boolean
    muteDuration: number
    echo?: boolean
  }
  welcomeMsg?: string
  goodbyeMsg?: string
  welcomeEnabled?: boolean
  goodbyeEnabled?: boolean
  levelLimit?: number
  leaveCooldown?: number
  /** 警告次数限制（覆盖全局设置） */
  warnLimit?: number
  dice?: {
    enabled: boolean
    lengthLimit: number
  }
  antiRepeat?: {
    enabled: boolean
    threshold: number
  }
  banme?: {
    enabled: boolean
    baseMin: number
    baseMax: number
    growthRate: number
    autoBan?: boolean
    jackpot: {
      enabled: boolean
      baseProb: number
      softPity: number
      hardPity: number
      upDuration: string
      loseDuration: string
    }
  }
  openai?: {
    enabled: boolean
    chatEnabled?: boolean
    translateEnabled?: boolean
    systemPrompt?: string
    translatePrompt?: string
  }
  antiRecall?: {
    enabled: boolean
    retentionDays?: number
    maxRecordsPerUser?: number
  }
  report?: {
    enabled: boolean
    autoProcess?: boolean
    includeContext?: boolean
    contextSize?: number
  }
}

// 警告记录
export interface WarnRecord {
  groups: {
    [guildId: string]: {
      count: number
      timestamp: number
    }
  }
}

// 黑名单记录
export type MemberBlacklistScopeType = 'global' | 'guild'
export type MemberBlacklistSource = 'admission_failure' | 'manual_admin' | 'kick_blacklist' | 'moderation_action' | 'migration_legacy_koishi' | 'migration_admission_failure'
export type MemberBlacklistReasonCode = 'admission_timeout_limit' | 'manual_blacklist' | 'manual_kick_blacklist' | 'violation_review_blacklist' | 'legacy_koishi_blacklist' | 'legacy_admission_blacklist'
export type MemberBlacklistActorType = 'system' | 'admin_user' | 'qq_operator' | 'service_account'
export type MemberBlacklistCreatedFrom = 'admission_worker' | 'qq_command' | 'koishi_console' | 'admin_console' | 'moderation_review' | 'migration_script'
export type MemberBlacklistReleaseReasonCode = 'manual_pardon' | 'release_only' | 'policy_expired_auto' | 'admission_appeal_passed' | 'migration_inverse'

export interface MemberBlacklistEntry {
  id: string
  platform: string
  subjectType: 'qq_user'
  subjectID: string
  scopeType: MemberBlacklistScopeType
  guildID?: string | null
  source: MemberBlacklistSource
  reasonCode: MemberBlacklistReasonCode
  reasonText: string
  createdByType: MemberBlacklistActorType
  createdByID: string
  createdFrom: MemberBlacklistCreatedFrom
  createdAt: string
  expiresAt?: string | null
  releasedAt?: string | null
  releasedByType?: string | null
  releasedByID?: string | null
  releaseReasonCode?: string | null
  releaseReason?: string | null
  metadata?: Record<string, unknown>
}

export interface MemberBlacklistListResult {
  list: MemberBlacklistEntry[]
  total: number
}

export interface MemberBlacklistCreateParams {
  platform: string
  subjectID: string
  scopeType: MemberBlacklistScopeType
  guildID?: string
  reasonText?: string
}

export interface MemberBlacklistReleaseParams {
  platform: string
  subjectID: string
  scopeType: MemberBlacklistScopeType
  guildID?: string
  releaseReasonCode?: MemberBlacklistReleaseReasonCode
  releaseReason?: string
}

// 订阅配置
export interface Subscription {
  type: 'group' | 'private'
  id: string
  name?: string
  avatar?: string
  features: {
    log?: boolean
    memberChange?: boolean
    muteExpire?: boolean
    blacklist?: boolean
    warning?: boolean
    antiRecall?: boolean
  }
}

// 仪表盘统计数据类型
export interface DashboardStats {
  totalGroups: number
  totalWarns: number
  totalBlacklisted: number
  totalSubscriptions: number
  timestamp: number
}

// 日志搜索参数
export interface LogSearchParams {
  startTime?: string | number
  endTime?: string | number
  command?: string
  userId?: string
  username?: string
  details?: string
  guildId?: string
  page?: number
  pageSize?: number
}

// 日志记录
export interface LogRecord {
  id: string
  timestamp: string
  userId: string
  username?: string
  userAuthority?: number
  guildId?: string
  guildName?: string
  channelId?: string
  platform: string
  command: string
  args: string[]
  options: Record<string, any>
  success: boolean
  error?: string
  executionTime: number
  result?: string
  messageId?: string
  isPrivate: boolean
}

// 日志响应
export interface LogResponse {
  list: LogRecord[]
  total: number
  page: number
  pageSize: number
}

// 聊天消息
export interface ChatMessage {
  id: string
  timestamp: number
  userId: string
  username: string
  avatar?: string
  content: string
  elements?: any[] // h elements
  platform: string
  guildId?: string
  guildName?: string
  guildAvatar?: string
  channelId: string
  channelName?: string
  selfId: string
}

// 权限相关类型
export interface Role {
  id: string
  name: string
  /** 角色别名（用于命令查找，没有时使用 name） */
  alias?: string
  color?: string
  priority: number
  permissions: string[]
  /** 角色生效的群组 ID 列表（空数组或 undefined 表示全局生效） */
  guildIds?: string[]
  /** 是否为内置角色（内置角色不可删除） */
  builtin?: boolean
}

export interface PermissionNode {
  id: string
  name: string
  description: string
  group?: string // 用于前端分组显示
}

// 扩展 @koishijs/client 的 Events 接口
declare module '@koishijs/client' {
  interface Events {
    // 权限管理 API
    'stuhelperGroupCenter/auth/role/list'(): Promise<Role[]>
    'stuhelperGroupCenter/auth/role/update'(params: { role: Role }): Promise<{ success: boolean }>
    'stuhelperGroupCenter/auth/role/delete'(params: { roleId: string }): Promise<{ success: boolean }>
    'stuhelperGroupCenter/auth/user/get'(params: { userId: string }): Promise<string[]>
    'stuhelperGroupCenter/auth/user/assign'(params: { userId: string, roleId: string }): Promise<{ success: boolean }>
    'stuhelperGroupCenter/auth/user/revoke'(params: { userId: string, roleId: string }): Promise<{ success: boolean }>
    'stuhelperGroupCenter/auth/permission/list'(): Promise<PermissionNode[]>

    // 群组配置 API
    'stuhelperGroupCenter/config/list'(): Promise<Record<string, GroupConfig>>
    'stuhelperGroupCenter/config/get'(guildId: string): Promise<GroupConfig | undefined>
    'stuhelperGroupCenter/config/update'(guildId: string, config: GroupConfig): Promise<{ success: boolean }>

    // 警告记录 API
    'stuhelperGroupCenter/warns/list'(): Promise<Record<string, WarnRecord>>
    'stuhelperGroupCenter/warns/get'(key: string): Promise<WarnRecord | undefined>
    'stuhelperGroupCenter/warns/clear'(key: string): Promise<{ success: boolean }>

    // 黑名单 API
    'stuhelperGroupCenter/blacklist/list'(): Promise<MemberBlacklistListResult>
    'stuhelperGroupCenter/blacklist/add'(params: MemberBlacklistCreateParams): Promise<MemberBlacklistEntry>
    'stuhelperGroupCenter/blacklist/remove'(params: MemberBlacklistReleaseParams): Promise<MemberBlacklistEntry>

    // 订阅 API
    'stuhelperGroupCenter/subscriptions/list'(): Promise<Subscription[]>
    'stuhelperGroupCenter/subscriptions/add'(subscription: Subscription): Promise<{ success: boolean }>
    'stuhelperGroupCenter/subscriptions/remove'(index: number): Promise<{ success: boolean }>

    // 统计 API
    'stuhelperGroupCenter/stats/dashboard'(): Promise<DashboardStats>

    // 日志 API
    'stuhelperGroupCenter/logs/search'(params: LogSearchParams): Promise<LogResponse>

    // 聊天 API
    'stuhelperGroupCenter/chat/send'(params: { channelId: string, content: string, platform?: string, guildId?: string }): Promise<{ success: boolean }>
  }
}
