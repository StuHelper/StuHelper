/**
 * WebUI API 客户端封装
 * 提供类型安全的 API 调用方法
 */

import { send } from '@koishijs/client'
import type { GroupConfig, WarnRecord, Subscription, Role, PermissionNode, RoleMember } from './types'

// 重新导出类型
export type { GroupConfig, WarnRecord, Subscription, Role, PermissionNode, RoleMember }

export type BlacklistScopeType = 'guild' | 'global'

export interface MemberBlacklistEntry {
  readonly id: string
  readonly platform: string
  readonly subjectID: string
  readonly scopeType: BlacklistScopeType
  readonly guildID?: string
  readonly source: string
  readonly reasonCode: string
  readonly reasonText: string
  readonly createdAt: string
}

export interface BlacklistListResult {
  readonly items: readonly MemberBlacklistEntry[]
}

export interface BlacklistCreateInput {
  readonly platform?: string
  readonly subjectID: string
  readonly scopeType: BlacklistScopeType
  readonly guildID?: string
  readonly reasonText?: string
}

// 仪表盘统计数据类型
export interface DashboardStats {
  totalGroups: number
  totalWarns: number
  totalBlacklisted: number
  totalSubscriptions: number
  timestamp: number
}

// API 响应类型
interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
}

// 通用调用封装
async function call<T>(event: keyof any, params?: any): Promise<T> {
  // @ts-ignore
  const result = await send(event, params) as ApiResponse<T>
  if (!result.success) {
    throw new Error(result.error || '请求失败')
  }
  // @ts-ignore
  return result.data
}

// 群组配置 API
export const configApi = {
  list: (fetchNames?: boolean) => call<Record<string, GroupConfig>>('stuhelperGroupCenter/config/list', { fetchNames }),
  get: (guildId: string) => call<GroupConfig | undefined>('stuhelperGroupCenter/config/get', { guildId }),
  update: (guildId: string, config: GroupConfig) => call<{ success: boolean }>('stuhelperGroupCenter/config/update', { guildId, config }),
  create: (guildId: string) => call<{ success: boolean }>('stuhelperGroupCenter/config/create', { guildId }),
  delete: (guildId: string) => call<{ success: boolean }>('stuhelperGroupCenter/config/delete', { guildId }),
  /** 重新从文件加载配置 */
  reload: () => call<{ success: boolean; count: number }>('stuhelperGroupCenter/config/reload'),
}

// 警告记录 API
export const warnsApi = {
  list: (fetchNames?: boolean) => call<any[]>('stuhelperGroupCenter/warns/list', { fetchNames }),
  get: (key: string) => call<WarnRecord | undefined>('stuhelperGroupCenter/warns/get', { key }),
  add: (guildId: string, userId: string) => call<{ success: boolean }>('stuhelperGroupCenter/warns/add', { guildId, userId }),
  clear: (key: string) => call<{ success: boolean }>('stuhelperGroupCenter/warns/clear', { key }),
  update: (key: string, count: number) => call<{ success: boolean }>('stuhelperGroupCenter/warns/update', { key, count }),
  /** 重新从文件加载警告数据 */
  reload: () => call<{ success: boolean }>('stuhelperGroupCenter/warns/reload'),
}

// 黑名单 API
export const blacklistApi = {
  list: () => call<BlacklistListResult>('stuhelperGroupCenter/blacklist/list'),
  add: (input: BlacklistCreateInput) => call<{ success: boolean }>('stuhelperGroupCenter/blacklist/add', input),
  remove: (id: string, releaseReason?: string) => call<{ success: boolean }>('stuhelperGroupCenter/blacklist/remove', { id, releaseReason }),
}

// 订阅 API
export const subscriptionApi = {
  list: (fetchNames?: boolean) => call<Subscription[]>('stuhelperGroupCenter/subscriptions/list', { fetchNames }),
  add: (subscription: Subscription) => call<{ success: boolean }>('stuhelperGroupCenter/subscriptions/add', { subscription }),
  remove: (index: number) => call<{ success: boolean }>('stuhelperGroupCenter/subscriptions/remove', { index }),
  update: (index: number, subscription: Subscription) => call<{ success: boolean }>('stuhelperGroupCenter/subscriptions/update', { index, subscription }),
}

export interface ModuleStatus {
  name: string
  description: string
  state: 'unloaded' | 'loading' | 'loaded' | 'error'
  error?: string
}

// 图表数据类型
export interface ChartTrendItem {
  date: string
  count: number
}

export interface ChartDistributionItem {
  command: string
  count: number
}

export interface ChartGuildRankItem {
  guildId: string
  count: number
  name?: string
}

export interface ChartUserRankItem {
  userId: string
  count: number
  name: string
}

export interface ChartData {
  trend: ChartTrendItem[]
  distribution: ChartDistributionItem[]
  successRate: { success: number; fail: number }
  guildRank: ChartGuildRankItem[]
  userRank: ChartUserRankItem[]
}

// 统计 API
export const statsApi = {
  dashboard: () => call<DashboardStats>('stuhelperGroupCenter/stats/dashboard'),
  modules: () => call<ModuleStatus[]>('stuhelperGroupCenter/stats/modules'),
  charts: (days?: number) => call<ChartData>('stuhelperGroupCenter/stats/charts', { days }),
}

// 日志 API
import type { LogSearchParams, LogResponse } from './types'
export const logsApi = {
  search: (params: LogSearchParams) => call<LogResponse>('stuhelperGroupCenter/logs/search', params),
}

// 全局设置 API
export const settingsApi = {
  get: () => call<any>('stuhelperGroupCenter/settings/get'),
  update: (settings: any) => call<{ success: boolean }>('stuhelperGroupCenter/settings/update', { settings }),
  reset: () => call<{ success: boolean }>('stuhelperGroupCenter/settings/reset'),
}

// 群成员类型
export interface GuildMember {
  id: string
  name: string
  avatar?: string
  isAdmin?: boolean
  isOwner?: boolean
  title?: string
  joinedAt?: number
}

// 聊天 API
export const chatApi = {
  send: (channelId: string, content: string, platform?: string, guildId?: string) =>
    call<{ success: boolean }>('stuhelperGroupCenter/chat/send', { channelId, content, platform, guildId }),
  /** 获取群信息 */
  getGuildInfo: (guildId: string) =>
    call<{ name?: string; avatar?: string }>('stuhelperGroupCenter/chat/guild-info', { guildId }),
  /** 获取用户信息 */
  getUserInfo: (userId: string) =>
    call<{ name?: string; avatar?: string }>('stuhelperGroupCenter/chat/user-info', { userId }),
  /** 获取群成员列表 */
  getGuildMembers: (guildId: string) =>
    call<{ members: GuildMember[]; total: number }>('stuhelperGroupCenter/chat/guild-members', { guildId }),
  /** 撤回消息 */
  recall: (channelId: string, messageId: string, platform?: string, guildId?: string) =>
    call<{ success: boolean }>('stuhelperGroupCenter/chat/recall', { channelId, messageId, platform, guildId }),
}

// 图片代理 API
export interface ImageProxyResult {
  success: boolean
  data?: {
    dataUrl?: string
    direct?: boolean
    source?: string
  }
  error?: string
}

export const imageApi = {
  /**
   * 获取图片（通过代理或本地缓存）
   * @param url 图片 URL
   * @param file 可选的文件标识（用于 OneBot get_image API）
   */
  fetch: async (url: string, file?: string): Promise<ImageProxyResult> => {
    try {
      // @ts-ignore - send 接受两个参数
      const result = await send('stuhelperGroupCenter/image/fetch', { url, file }) as ImageProxyResult
      return result
    } catch (e: any) {
      return { success: false, error: e.message || '图片加载失败' }
    }
  },
}

// 缓存管理 API
export interface CacheStats {
  guilds: number
  users: number
  members: number
  lastFullRefresh: number
  lastFullRefreshTime: string
}

export const cacheApi = {
  /** 获取缓存统计信息 */
  stats: () => call<CacheStats>('stuhelperGroupCenter/cache/stats'),
  /** 强制刷新所有缓存 */
  refresh: () => call<{ success: boolean; stats: CacheStats }>('stuhelperGroupCenter/cache/refresh'),
  /** 清空所有缓存 */
  clear: () => call<{ success: boolean }>('stuhelperGroupCenter/cache/clear'),
  /** 按需获取名称（会触发缓存） */
  fetchName: (type: 'guild' | 'user' | 'member', guildId?: string, userId?: string) =>
    call<{ name?: string; nick?: string; avatar?: string }>('stuhelperGroupCenter/cache/fetch-name', { type, guildId, userId }),
}

// 权限管理 API
export const authApi = {
  getRoles: () => call<Role[]>('stuhelperGroupCenter/auth/role/list'),
  updateRole: (role: Role) => call<{ success: boolean }>('stuhelperGroupCenter/auth/role/update', { role }),
  deleteRole: (roleId: string) => call<{ success: boolean }>('stuhelperGroupCenter/auth/role/delete', { roleId }),
  getUserRoles: (userId: string) => call<string[]>('stuhelperGroupCenter/auth/user/get', { userId }),
  assignRole: (userId: string, roleId: string) => call<{ success: boolean }>('stuhelperGroupCenter/auth/user/assign', { userId, roleId }),
  revokeRole: (userId: string, roleId: string) => call<{ success: boolean }>('stuhelperGroupCenter/auth/user/revoke', { userId, roleId }),
  getPermissions: () => call<PermissionNode[]>('stuhelperGroupCenter/auth/permission/list'),
  getRoleMembers: (roleId: string, fetchNames?: boolean) => call<RoleMember[]>('stuhelperGroupCenter/auth/role/members', { roleId, fetchNames }),
  /** 批量导入成员到角色 */
  importMembers: (roleId: string, userIds: string[]) => call<{ success: boolean; imported: number }>('stuhelperGroupCenter/auth/role/import-members', { roleId, userIds }),
  /** 获取指定 authority 等级的用户列表 */
  getUsersByAuthority: (authority: number) => call<RoleMember[]>('stuhelperGroupCenter/auth/users-by-authority', { authority }),
  /** 获取指定群的管理员列表 */
  getGuildAdmins: (guildId: string) => call<RoleMember[]>('stuhelperGroupCenter/auth/guild-admins', { guildId }),
}
