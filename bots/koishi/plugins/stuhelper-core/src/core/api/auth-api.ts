import type { Bot } from 'koishi'

import type { Role } from '../../types'
import { normalizeManagedRoleInput, requireAssignableRole } from './auth-management'
import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  type ConsoleGuildScope,
} from './console-guild-scope'
import { assertReadableRole, filterRoleIds, filterRoles } from './scope-filters'
import { isGuildAdminMember, type GuildAdminMember } from '../services/auth-guild-admin'

const MIN_AUTHORITY_LEVEL = 1
const MAX_AUTHORITY_LEVEL = 5
const QQ_PLATFORMS = new Set(['onebot', 'red', 'qq'])

export function registerAuthAPI(api: WebSocketAPIContext): void {
  registerRoleAPI(api)
  registerUserRoleAPI(api)
  registerAuthQueryAPI(api)
  api.addAuthorityListener('stuhelperGroupCenter/auth/permission/list', async () => {
    return success(api.service.auth.getPermissions())
  })
}

function registerRoleAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/auth/role/list', async function () {
    const scope = await api.resolveConsoleScope(this)
    return success(filterRoles(api.service.auth.getRoles(), scope))
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/role/update', async function (params: { role: Role }) {
    const scope = await api.resolveConsoleScope(this)
    const role = normalizeManagedRoleInput(api.service.auth, scope, params.role)
    await api.service.auth.saveRole(role)
    await api.service.data.authRoles.flush()
    return success({ success: true })
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/role/delete', async function (params: { roleId: string }) {
    const scope = await api.resolveConsoleScope(this)
    requireAssignableRole(api.service.auth, scope, params.roleId)
    await api.service.auth.deleteRole(params.roleId)
    await api.service.data.authRoles.flush()
    await api.service.data.authUsers.flush()
    return success({ success: true })
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/role/members', async function (params: RoleMembersParams) {
    return handleRoleMembers(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/role/import-members', async function (params: ImportMembersParams) {
    return handleImportMembers(api, this, params)
  })
}

function registerUserRoleAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/auth/user/get', async function (params: { userId: string }) {
    const scope = await api.resolveConsoleScope(this)
    return success(filterRoleIds(api.service.auth.getUserRoleIds(params.userId), api.service.auth.getRoles(), scope))
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/user/assign', async function (params: UserRoleParams) {
    const scope = await api.resolveConsoleScope(this)
    requireAssignableRole(api.service.auth, scope, params.roleId)
    await api.service.auth.assignRole(params.userId, params.roleId)
    await api.service.data.authUsers.flush()
    return success({ success: true })
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/user/revoke', async function (params: UserRoleParams) {
    const scope = await api.resolveConsoleScope(this)
    requireAssignableRole(api.service.auth, scope, params.roleId)
    await api.service.auth.revokeRole(params.userId, params.roleId)
    await api.service.data.authUsers.flush()
    return success({ success: true })
  })
}

function registerAuthQueryAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/auth/users-by-authority', async function (params: AuthorityQueryParams) {
    return handleUsersByAuthority(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/auth/guild-admins', async function (params: { guildId: string }) {
    return handleGuildAdmins(api, this, params.guildId)
  })
}

interface RoleMembersParams {
  readonly roleId: string
  readonly fetchNames?: boolean
}

interface ImportMembersParams {
  readonly roleId: string
  readonly userIds: string[]
}

interface UserRoleParams {
  readonly userId: string
  readonly roleId: string
}

interface AuthorityQueryParams {
  readonly authority: number
}

interface RoleMemberProfile {
  readonly id: string
  readonly name: string
  readonly avatar: string
}

interface AuthorityUser {
  readonly id: number
  readonly name?: string
}

interface UserBinding {
  readonly aid: number
  readonly platform: string
  readonly pid: string
}

type SelectedUserBinding = Pick<UserBinding, 'platform' | 'pid'>
type GuildMemberListBot = Pick<Bot, 'platform' | 'getGuildMemberList'>
type GuildAdminListMember = GuildAdminMember & {
  readonly userId?: string
}

async function handleRoleMembers(api: WebSocketAPIContext, client: unknown, params: RoleMembersParams) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertReadableRole(scope, api.service.auth.getRoles(), params.roleId)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取角色成员失败')
  }

  const userIds = api.service.auth.getRoleMembers(params.roleId)
  if (!params.fetchNames) {
    return success(userIds.map((id) => ({ id, name: '', avatar: '' })))
  }
  return success(buildRoleMemberProfiles(api, userIds))
}

function buildRoleMemberProfiles(api: WebSocketAPIContext, userIds: string[]) {
  const cacheData = api.service.cache.getCachedData()
  return userIds.map((userId) => {
    const cached = cacheData.users[userId]
    return {
      id: userId,
      name: cached?.name || '',
      avatar: cached?.avatar || `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`,
    }
  })
}

async function handleImportMembers(api: WebSocketAPIContext, client: unknown, params: ImportMembersParams) {
  try {
    validateImportMembersParams(params)
    const scope = await api.resolveConsoleScope(client)
    requireAssignableRole(api.service.auth, scope, params.roleId)
    if (api.service.auth.isBuiltinRole(params.roleId)) {
      return error('内置角色由系统自动分配，不支持手动添加成员')
    }
    const imported = await importRoleMembers(api, params)
    return success({ success: true, imported })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '导入失败')
  }
}

function validateImportMembersParams(params: ImportMembersParams): void {
  if (!params.roleId || !params.userIds || !Array.isArray(params.userIds)) {
    throw new Error('无效的参数')
  }
}

async function importRoleMembers(api: WebSocketAPIContext, params: ImportMembersParams) {
  let imported = 0
  for (const userId of params.userIds) {
    if (!userId || typeof userId !== 'string') continue
    await api.service.auth.assignRole(userId.trim(), params.roleId)
    imported++
  }
  await api.service.data.authUsers.flush()
  return imported
}

async function handleUsersByAuthority(api: WebSocketAPIContext, client: unknown, params: AuthorityQueryParams) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'authority user query')
    validateAuthorityLevel(params.authority)
    const users = await api.ctx.database.get('user', { authority: params.authority })
    if (users.length === 0) return success([])
    const bindings = await api.ctx.database.get('binding', { aid: { $in: users.map((user) => user.id) } })
    return success(buildAuthorityMembers(api, users, bindings))
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('获取权限用户列表失败:', cause)
    return error(cause instanceof Error ? cause.message : '获取用户列表失败')
  }
}

function validateAuthorityLevel(authority: number): void {
  if (typeof authority !== 'number' || authority < MIN_AUTHORITY_LEVEL || authority > MAX_AUTHORITY_LEVEL) {
    throw new Error('无效的权限等级')
  }
}

function buildAuthorityMembers(
  api: WebSocketAPIContext,
  users: readonly AuthorityUser[],
  bindings: readonly UserBinding[],
): RoleMemberProfile[] {
  const aidToBinding = selectPreferredBindings(bindings)
  const cacheData = api.service.cache.getCachedData()
  return users.filter((user) => aidToBinding[user.id]).map((user) => {
    const binding = aidToBinding[user.id]
    const userId = binding.pid
    const cached = cacheData.users[userId]
    return {
      id: userId,
      name: cached?.name || user.name || '',
      avatar: cached?.avatar || (QQ_PLATFORMS.has(binding.platform) ? `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640` : ''),
    }
  })
}

function selectPreferredBindings(bindings: readonly UserBinding[]): Record<number, SelectedUserBinding> {
  const aidToBinding: Record<number, SelectedUserBinding> = {}
  for (const binding of bindings) {
    const existing = aidToBinding[binding.aid]
    const isQQPlatform = QQ_PLATFORMS.has(binding.platform)
    if (!existing || (isQQPlatform && !QQ_PLATFORMS.has(existing.platform))) {
      aidToBinding[binding.aid] = { platform: binding.platform, pid: binding.pid }
    }
  }
  return aidToBinding
}

async function handleGuildAdmins(api: WebSocketAPIContext, client: unknown, guildId: string) {
  try {
    validateGuildId(guildId)
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, guildId, 'guild admin query')
    return await fetchGuildAdmins(api, scope, guildId)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取群管理员列表失败')
  }
}

function validateGuildId(guildId: string): void {
  if (!guildId) {
    throw new Error('缺少群号')
  }
}

async function fetchGuildAdmins(api: WebSocketAPIContext, scope: ConsoleGuildScope, guildId: string) {
  assertConsoleGuildAccess(scope, guildId, 'guild admin query')
  for (const bot of api.ctx.bots) {
    try {
      const members = await fetchAllGuildMembers(bot, guildId)
      return success(members.filter(isGuildAdminMember).map((member) => formatGuildAdmin(bot, member)))
    } catch (cause) {
      api.ctx.logger('stuhelperGroupCenter').warn('获取群管理员列表失败:', cause)
    }
  }
  return error('无法获取群管理员列表')
}

async function fetchAllGuildMembers(bot: GuildMemberListBot, guildId: string): Promise<GuildAdminListMember[]> {
  const members: GuildAdminListMember[] = []
  let next: string | undefined
  do {
    const result = await bot.getGuildMemberList(guildId, next)
    if (result.data) members.push(...result.data)
    next = result.next
  } while (next)
  return members
}

function formatGuildAdmin(bot: Pick<Bot, 'platform'>, member: GuildAdminListMember): RoleMemberProfile {
  const userId = member.user?.id || member.userId || ''
  let avatar = member.user?.avatar || member.avatar
  if (!avatar && bot.platform && QQ_PLATFORMS.has(bot.platform)) {
    avatar = `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
  }
  return {
    id: userId,
    name: member.nick || member.user?.nick || member.user?.name || member.name || userId,
    avatar: avatar || '',
  }
}
