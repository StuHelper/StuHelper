import type { Session, Context } from 'koishi'
import type { Puppeteer } from 'koishi-plugin-puppeteer'
import type { Status } from '@koishijs/plugin-status'

import type {
  Subscription,
  GroupConfig,
  Config,
  WarnRecord,
  BlacklistRecord,
  PermissionNode,
  Role,
} from './types'
import type {
  DashboardPageData,
  IdentityPageData,
  ReviewPageData,
  ConfigGovernancePageData,
  ReviewWorkItem,
  ModuleStateSnapshot,
} from './core/services/page-types'

declare module 'koishi' {
  interface Context {
    puppeteer: Puppeteer
    status: Status
  }

  interface Events {
    send(session: Session): void
  }
}

interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
}

interface RoleMemberSnapshot {
  id: string
  name: string
  avatar: string
}

interface WarnListItem {
  key: string
  guildId: string
  userId: string
  guildName: string
  guildAvatar: string
  userName: string
  userAvatar: string
  count: number
  timestamp: number
}

interface DashboardStatsPayload {
  totalGroups: number
  totalWarns: number
  totalBlacklisted: number
  totalSubscriptions: number
  version: string
  timestamp: number
}

interface ChartStatsPayload {
  trend: Array<{ date: string; count: number }>
  distribution: Array<{ command: string; count: number }>
  successRate: { success: number; fail: number }
  guildRank: Array<{ guildId: string; count: number; name: string }>
  userRank: Array<{ userId: string; count: number; name: string }>
}

interface CacheFetchNameParams {
  type: 'guild' | 'user' | 'member'
  guildId?: string
  userId?: string
}

interface ChatGuildMember {
  id: string
  name: string
  avatar?: string
  isAdmin: boolean
  isOwner: boolean
  title: string
  joinedAt?: number | string | Date
}

interface ChatMessagePayload {
  id: string
  timestamp: number
  userId: string
  username: string
  avatar?: string
  content: string
  elements: unknown[]
  platform?: string
  guildId?: string
  guildName?: string
  guildAvatar?: string
  channelId?: string
  channelName?: string
  selfId?: string
}

declare module '@koishijs/console' {
  interface Events {
    'stuhelperGroupCenter/config/reload'(): Promise<ApiResponse<{ success: boolean; count: number }>>
    'stuhelperGroupCenter/config/list'(params?: { fetchNames?: boolean }): Promise<ApiResponse<Record<string, GroupConfig>>>
    'stuhelperGroupCenter/config/get'(params: { guildId: string }): Promise<ApiResponse<GroupConfig | undefined>>
    'stuhelperGroupCenter/config/update'(params: { guildId: string; config: GroupConfig }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/config/create'(params: { guildId: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/config/delete'(params: { guildId: string }): Promise<ApiResponse<{ success: boolean }>>

    'stuhelperGroupCenter/auth/role/list'(): Promise<ApiResponse<Role[]>>
    'stuhelperGroupCenter/auth/role/update'(params: { role: Role }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/auth/role/delete'(params: { roleId: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/auth/user/get'(params: { userId: string }): Promise<ApiResponse<string[]>>
    'stuhelperGroupCenter/auth/role/members'(params: { roleId: string; fetchNames?: boolean }): Promise<ApiResponse<RoleMemberSnapshot[]>>
    'stuhelperGroupCenter/auth/user/assign'(params: { userId: string; roleId: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/auth/user/revoke'(params: { userId: string; roleId: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/auth/role/import-members'(params: { roleId: string; userIds: string[] }): Promise<ApiResponse<{ success: boolean; imported?: number }>>
    'stuhelperGroupCenter/auth/users-by-authority'(params: { authority: number }): Promise<ApiResponse<RoleMemberSnapshot[]>>
    'stuhelperGroupCenter/auth/guild-admins'(params: { guildId: string }): Promise<ApiResponse<RoleMemberSnapshot[]>>
    'stuhelperGroupCenter/auth/permission/list'(): Promise<ApiResponse<PermissionNode[]>>

    'stuhelperGroupCenter/warns/reload'(): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/warns/list'(params?: { fetchNames?: boolean }): Promise<ApiResponse<WarnListItem[]>>
    'stuhelperGroupCenter/warns/get'(params: { key: string }): Promise<ApiResponse<WarnRecord[string] | null | undefined>>
    'stuhelperGroupCenter/warns/update'(params: { key: string; count: number }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/warns/add'(params: { guildId: string; userId: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/warns/clear'(params: { key: string }): Promise<ApiResponse<{ success: boolean }>>

    'stuhelperGroupCenter/blacklist/list'(): Promise<ApiResponse<Record<string, BlacklistRecord>>>
    'stuhelperGroupCenter/blacklist/add'(params: { userId: string; record: BlacklistRecord }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/blacklist/remove'(params: { userId: string }): Promise<ApiResponse<{ success: boolean }>>

    'stuhelperGroupCenter/subscriptions/list'(params?: { fetchNames?: boolean }): Promise<ApiResponse<Array<Subscription & { name?: string; avatar?: string }>>>
    'stuhelperGroupCenter/subscriptions/add'(params: { subscription: Subscription }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/subscriptions/remove'(params: { index: number }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/subscriptions/update'(params: { index: number; subscription: Subscription }): Promise<ApiResponse<{ success: boolean }>>

    'stuhelperGroupCenter/stats/modules'(): Promise<ApiResponse<ModuleStateSnapshot[]>>
    'stuhelperGroupCenter/stats/dashboard'(): Promise<ApiResponse<DashboardStatsPayload>>
    'stuhelperGroupCenter/stats/charts'(params?: { days?: number }): Promise<ApiResponse<ChartStatsPayload>>

    'stuhelperGroupCenter/logs/search'(params: {
      startTime?: string | number
      endTime?: string | number
      command?: string
      userId?: string
      username?: string
      details?: string
      guildId?: string
      page?: number
      pageSize?: number
    }): Promise<ApiResponse<{
      list: unknown[]
      total: number
      page: number
      pageSize: number
    }>>

    'stuhelperGroupCenter/settings/get'(): Promise<ApiResponse<Config>>
    'stuhelperGroupCenter/settings/update'(params: { settings: Config }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/settings/reset'(): Promise<ApiResponse<{ success: boolean }>>

    'stuhelperGroupCenter/cache/stats'(): Promise<ApiResponse<unknown>>
    'stuhelperGroupCenter/cache/refresh'(): Promise<ApiResponse<{ success: boolean; stats: unknown }>>
    'stuhelperGroupCenter/cache/clear'(): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/cache/fetch-name'(params: CacheFetchNameParams): Promise<ApiResponse<{ name: string; avatar?: string; nick?: string }>>

    'stuhelperGroupCenter/chat/guild-members'(params: { guildId: string }): Promise<ApiResponse<{ members: ChatGuildMember[]; total: number }>>
    'stuhelperGroupCenter/chat/guild-info'(params: { guildId: string }): Promise<ApiResponse<{ name: string; avatar?: string }>>
    'stuhelperGroupCenter/chat/user-info'(params: { userId: string }): Promise<ApiResponse<{ name: string; avatar?: string }>>
    'stuhelperGroupCenter/chat/send'(params: { channelId: string; content: string; platform?: string; guildId?: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/chat/recall'(params: { channelId: string; messageId: string; platform?: string }): Promise<ApiResponse<{ success: boolean }>>
    'stuhelperGroupCenter/chat/message'(payload: ChatMessagePayload): void

    'stuhelperGroupCenter/image/fetch'(params: { url: string; file?: string }): Promise<ApiResponse<{
      dataUrl: string
      hash: string
      mimeType: string
      source: string
    }>>

    'stuhelperGroupCenter/page/dashboard'(): Promise<DashboardPageData>
    'stuhelperGroupCenter/page/identity'(): Promise<IdentityPageData>
    'stuhelperGroupCenter/page/review'(): Promise<ReviewPageData>
    'stuhelperGroupCenter/page/config-governance'(): Promise<ConfigGovernancePageData>

    'stuhelperGroupCenter/action/review'(input: {
      reviewId: string
      action: 'execute' | 'reject'
      note?: string
    }): Promise<string>
    'stuhelperGroupCenter/action/work-item'(input: {
      kind: ReviewWorkItem['kind']
      itemId: string
      action:
        | 'execute'
        | 'reject'
        | 'approve'
        | 'deny'
        | 'defer'
        | 'dismiss'
        | 'escalate'
        | 'create-review'
      note?: string
    }): Promise<string>
    'stuhelperGroupCenter/action/save-command-policy'(input: {
      commandId: string
      roles: string[]
      minAuthority: number
    }): Promise<string>
    'stuhelperGroupCenter/action/save-guard-template'(input: {
      id: string
      name: string
      muteDurationSeconds: number
      kickAfterMinutes: number
      reminderTemplate: string
      exemptUsers: string[]
      enabled: boolean
    }): Promise<string>
    'stuhelperGroupCenter/action/save-guard-binding'(input: {
      platform: string
      guildId: string
      templateId: string
      enabled: boolean
      note?: string | null
    }): Promise<string>
  }
}
