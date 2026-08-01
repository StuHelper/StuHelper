/**
 * WebUI API 客户端封装
 * 提供类型安全的 API 调用方法
 */

import { send } from '@koishijs/client'
import type {
  GroupConfig,
  PluginSettings,
  WarnRecord,
  WarnListItem,
  MemberBlacklistCreateParams,
  MemberBlacklistEntry,
  MemberBlacklistListResult,
  MemberBlacklistReleaseParams,
  Subscription,
  Role,
  PermissionNode,
  RoleMember,
} from './types'

// 重新导出类型
export type {
  GroupConfig,
  PluginSettings,
  WarnRecord,
  WarnListItem,
  MemberBlacklistCreateParams,
  MemberBlacklistEntry,
  MemberBlacklistListResult,
  MemberBlacklistReleaseParams,
  Subscription,
  Role,
  PermissionNode,
  RoleMember,
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

type ConsoleSend = (event: string, params?: unknown) => Promise<unknown>

const sendConsole = send as ConsoleSend

// 通用调用封装
async function call<T>(event: string, params?: unknown): Promise<T> {
  const result = await sendConsole(event, params)
  return unwrapApiResponse<T>(result)
}

function unwrapApiResponse<T>(result: unknown): T {
  if (!isApiResponse<T>(result)) {
    throw new Error('请求失败')
  }
  if (!result.success) {
    throw new Error(result.error || '请求失败')
  }
  return result.data as T
}

function isApiResponse<T>(value: unknown): value is ApiResponse<T> {
  if (!isRecord(value) || typeof value.success !== 'boolean') {
    return false
  }
  return value.error === undefined || typeof value.error === 'string'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
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
  list: (fetchNames?: boolean) => call<WarnListItem[]>('stuhelperGroupCenter/warns/list', { fetchNames }),
  get: (key: string) => call<WarnRecord | undefined>('stuhelperGroupCenter/warns/get', { key }),
  add: (guildId: string, userId: string) => call<{ success: boolean }>('stuhelperGroupCenter/warns/add', { guildId, userId }),
  clear: (key: string) => call<{ success: boolean }>('stuhelperGroupCenter/warns/clear', { key }),
  update: (key: string, count: number) => call<{ success: boolean }>('stuhelperGroupCenter/warns/update', { key, count }),
  /** 重新从文件加载警告数据 */
  reload: () => call<{ success: boolean }>('stuhelperGroupCenter/warns/reload'),
}

// 黑名单 API
export const blacklistApi = {
  list: () => call<MemberBlacklistListResult>('stuhelperGroupCenter/blacklist/list'),
  add: (input: MemberBlacklistCreateParams) =>
    call<MemberBlacklistEntry>('stuhelperGroupCenter/blacklist/add', input),
  remove: (input: MemberBlacklistReleaseParams) =>
    call<MemberBlacklistEntry>('stuhelperGroupCenter/blacklist/remove', input),
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

export interface BindingMessageSettings {
  directOnly: string
  missingCode: string
  successVerified: string
  successUnverified: string
  unavailable: string
  invalidCode: string
  unauthorized: string
  conflict: string
  notConfigured: string
}

export interface BindingRuntimeSettings {
  command: string
  messages: BindingMessageSettings
}

export interface AdminMessageSettings {
  guardStatusCommandDescription: string
  guardWarningCommandDescription: string
  guardReviewListCommandDescription: string
  guardBatchMuteCommandDescription: string
  guardKickReviewCommandDescription: string
  guardBlockReviewCommandDescription: string
  freshmanViewCommandDescription: string
  freshmanApproveCommandDescription: string
  freshmanRejectCommandDescription: string
  freshmanBlacklistReleaseCommandDescription: string
  commandAccessDenied: string
  adminCommandsDisabled: string
  guardGuildContextRequired: string
  guardWarningMissingContext: string
  guardBatchMuteGroupOnly: string
  guardBatchMuteInvalidPayload: string
  guardBatchMuteNoTargets: string
  guardBatchMuteSuccess: string
  guardBatchMuteEventSummary: string
  guardBatchMuteEventReason: string
  guardReviewRequestMissingArgs: string
  guardReviewRequestSuccess: string
  guardReviewRequestEventSummary: string
  guardReviewKickActionLabel: string
  guardReviewKickAndBlockActionLabel: string
  guardPendingMembersEmpty: string
  guardPendingMembersHeader: string
  guardPendingMemberLine: string
  guardWarningCounterEmpty: string
  guardWarningCounterNoReason: string
  guardWarningCounterSummary: string
  guardPendingReviewsEmpty: string
  guardPendingReviewsHeader: string
  guardPendingReviewLine: string
  freshmanManagementGroupOnly: string
  freshmanMissingApplicationID: string
  freshmanApproveInvalidFormat: string
  freshmanApproveInvalidExtension: string
  freshmanRejectInvalidFormat: string
  freshmanBlacklistReleaseInvalidFormat: string
  freshmanApproveSuccess: string
  freshmanApproveSuccessWithExtension: string
  freshmanRejectSuccess: string
  freshmanBlacklistScopeGlobal: string
  freshmanBlacklistScopeGuild: string
  freshmanBlacklistReleaseSuccess: string
  freshmanApplicationSummary: string
  freshmanApplicationDepartmentFallback: string
  freshmanApplicationExpiryFallback: string
  freshmanCommandFailed: string
  freshmanOperatorQQUnbound: string
  freshmanOperatorForbidden: string
  freshmanManagementGuildForbidden: string
  freshmanBackendForbidden: string
}

export interface AdminRuntimeSettings {
  messages: AdminMessageSettings
}

export interface GroupGuardFunSettings {
  diceSides: number
  muteLotteryBaseSeconds: number
  muteLotteryMaxSeconds: number
  muteLotteryPityThreshold: number
  muteLotteryPitySeconds: number
}

export interface GroupGuardModerationSettings {
  repeatThreshold: number
  repeatWindowSize: number
  warningThresholdExpression: string
  defaultMuteSeconds: number
  antiRecallNotify: boolean
}

export interface GroupGuardBehaviorSettings {
  fun: GroupGuardFunSettings
  moderation: GroupGuardModerationSettings
}

export interface GroupGuardAISettings {
  enabled: boolean
  endpoint: string
  model: string
  apiKeyConfigured: boolean
  apiKeyMasked: string
}

export interface GroupGuardAISettingsUpdate {
  enabled: boolean
  endpoint: string
  model: string
  newApiKey?: string
  clearApiKey?: boolean
}

export interface GroupGuardMessageSettings {
  messages: Record<string, string>
}

export type KeywordRuleMatchMode = 'includes' | 'regex'
export type KeywordRuleAction = 'warn' | 'delete' | 'mute' | 'review'

export interface KeywordRule {
  id: string
  guildId: string
  pattern: string
  matchMode: KeywordRuleMatchMode
  action: KeywordRuleAction
  enabled: boolean
  muteSeconds: number
  note: string | null
  createdAt?: string
  updatedAt?: string
}

export interface KeywordRuleInput {
  id: string
  guildId: string
  pattern: string
  matchMode: KeywordRuleMatchMode
  action: KeywordRuleAction
  enabled: boolean
  muteSeconds: number
  note: string | null
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
  get: () => call<PluginSettings>('stuhelperGroupCenter/settings/get'),
  update: (settings: PluginSettings) => call<{ success: boolean }>('stuhelperGroupCenter/settings/update', { settings }),
  reset: () => call<{ success: boolean }>('stuhelperGroupCenter/settings/reset'),
}

export const adminSettingsApi = {
  get: () => call<AdminRuntimeSettings>('stuhelperGroupCenter/admin-settings/get'),
  update: (settings: AdminRuntimeSettings) =>
    call<AdminRuntimeSettings>('stuhelperGroupCenter/admin-settings/update', { settings }),
  reset: () => call<AdminRuntimeSettings>('stuhelperGroupCenter/admin-settings/reset'),
}

export const bindingSettingsApi = {
  get: () => call<BindingRuntimeSettings>('stuhelperGroupCenter/binding-settings/get'),
  update: (settings: BindingRuntimeSettings) =>
    call<BindingRuntimeSettings>('stuhelperGroupCenter/binding-settings/update', { settings }),
  reset: () => call<BindingRuntimeSettings>('stuhelperGroupCenter/binding-settings/reset'),
}

export const groupGuardAISettingsApi = {
  get: () => call<GroupGuardAISettings>('stuhelperGroupCenter/group-guard-ai-settings/get'),
  update: (settings: GroupGuardAISettingsUpdate) =>
    call<GroupGuardAISettings>('stuhelperGroupCenter/group-guard-ai-settings/update', { settings }),
  reset: () => call<GroupGuardAISettings>('stuhelperGroupCenter/group-guard-ai-settings/reset'),
}

export const groupGuardBehaviorSettingsApi = {
  get: () => call<GroupGuardBehaviorSettings>('stuhelperGroupCenter/group-guard-behavior-settings/get'),
  update: (settings: GroupGuardBehaviorSettings) =>
    call<GroupGuardBehaviorSettings>('stuhelperGroupCenter/group-guard-behavior-settings/update', { settings }),
  reset: () => call<GroupGuardBehaviorSettings>('stuhelperGroupCenter/group-guard-behavior-settings/reset'),
}

export const groupGuardMessageSettingsApi = {
  get: () => call<GroupGuardMessageSettings>('stuhelperGroupCenter/group-guard-message-settings/get'),
  update: (settings: GroupGuardMessageSettings) =>
    call<GroupGuardMessageSettings>('stuhelperGroupCenter/group-guard-message-settings/update', { settings }),
  reset: () => call<GroupGuardMessageSettings>('stuhelperGroupCenter/group-guard-message-settings/reset'),
}

export const keywordRulesApi = {
  list: () => call<KeywordRule[]>('stuhelperGroupCenter/keyword-rules/list'),
  upsert: (rule: KeywordRuleInput) =>
    call<KeywordRule>('stuhelperGroupCenter/keyword-rules/upsert', { rule }),
  delete: (id: string) =>
    call<{ success: boolean }>('stuhelperGroupCenter/keyword-rules/delete', { id }),
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
      const result = await sendConsole('stuhelperGroupCenter/image/fetch', { url, file })
      return readImageProxyResult(result)
    } catch (cause) {
      return { success: false, error: cause instanceof Error ? cause.message : '图片加载失败' }
    }
  },
}

function readImageProxyResult(result: unknown): ImageProxyResult {
  if (!isApiResponse<ImageProxyResult['data']>(result)) {
    return { success: false, error: '图片加载失败' }
  }
  if (!result.success) {
    return { success: false, error: result.error || '图片加载失败' }
  }
  return { success: true, data: normalizeImageProxyData(result.data) }
}

function normalizeImageProxyData(data: ImageProxyResult['data'] | undefined): ImageProxyResult['data'] | undefined {
  if (!isRecord(data)) {
    return undefined
  }
  return {
    dataUrl: typeof data.dataUrl === 'string' ? data.dataUrl : undefined,
    direct: data.direct === true,
    source: typeof data.source === 'string' ? data.source : undefined,
  }
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
  reorderRoles: (sourceRoleId: string, targetRoleId: string) =>
    call<{ success: boolean }>('stuhelperGroupCenter/auth/role/reorder', { sourceRoleId, targetRoleId }),
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
