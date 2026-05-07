import type { Role } from '../../types'
import {
  normalizeManagedRoleInput,
  requireAssignableRole,
} from './auth-management'
import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'
import { error, success } from './api-response'
import {
  assertReadableRole,
  filterRoleIds,
  filterRoles,
  type WebSocketAPIContext,
} from './websocket-api-context'

const QQ_PLATFORMS = ['onebot', 'red', 'qq']

export function registerAuthAPI(api: WebSocketAPIContext) {
  const { ctx, service, addAuthorityListener, resolveConsoleScope } = api

  addAuthorityListener('stuhelperGroupCenter/auth/role/list', async function () {
    const scope = await resolveConsoleScope(this)
    return success(filterRoles(service.auth.getRoles(), scope))
  })

  addAuthorityListener('stuhelperGroupCenter/auth/role/update', async function (params: { role: Role }) {
    const scope = await resolveConsoleScope(this)
    const role = normalizeManagedRoleInput(service.auth, scope, params.role)
    await service.auth.saveRole(role)
    await service.data.authRoles.flush()
    return success({ success: true })
  })

  addAuthorityListener('stuhelperGroupCenter/auth/role/delete', async function (params: { roleId: string }) {
    const scope = await resolveConsoleScope(this)
    requireAssignableRole(service.auth, scope, params.roleId)
    await service.auth.deleteRole(params.roleId)
    await service.data.authRoles.flush()
    await service.data.authUsers.flush()
    return success({ success: true })
  })

  addAuthorityListener('stuhelperGroupCenter/auth/user/get', async function (params: { userId: string }) {
    const scope = await resolveConsoleScope(this)
    return success(filterRoleIds(service.auth.getUserRoleIds(params.userId), service.auth.getRoles(), scope))
  })

  addAuthorityListener('stuhelperGroupCenter/auth/role/members', async function (params: { roleId: string, fetchNames?: boolean }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertReadableRole(service.auth.getRoles(), scope, params.roleId)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取角色成员失败')
    }
    return success(resolveRoleMembers(service, params.roleId, !!params.fetchNames))
  })

  addAuthorityListener('stuhelperGroupCenter/auth/user/assign', async function (params: { userId: string, roleId: string }) {
    const scope = await resolveConsoleScope(this)
    requireAssignableRole(service.auth, scope, params.roleId)
    await service.auth.assignRole(params.userId, params.roleId)
    await service.data.authUsers.flush()
    return success({ success: true })
  })

  addAuthorityListener('stuhelperGroupCenter/auth/user/revoke', async function (params: { userId: string, roleId: string }) {
    const scope = await resolveConsoleScope(this)
    requireAssignableRole(service.auth, scope, params.roleId)
    await service.auth.revokeRole(params.userId, params.roleId)
    await service.data.authUsers.flush()
    return success({ success: true })
  })

  addAuthorityListener('stuhelperGroupCenter/auth/role/import-members', async function (params: { roleId: string, userIds: string[] }) {
    return importRoleMembers(api, params, this)
  })

  addAuthorityListener('stuhelperGroupCenter/auth/users-by-authority', async function (params: { authority: number }) {
    return listUsersByAuthority(api, params, this)
  })

  addAuthorityListener('stuhelperGroupCenter/auth/guild-admins', async function (params: { guildId: string }) {
    return listGuildAdmins(api, params, this)
  })

  addAuthorityListener('stuhelperGroupCenter/auth/permission/list', async () => {
    return success(service.auth.getPermissions())
  })
}

function resolveRoleMembers(
  service: WebSocketAPIContext['service'],
  roleId: string,
  fetchNames: boolean,
) {
  const userIds = service.auth.getRoleMembers(roleId)
  if (!fetchNames) {
    return userIds.map(id => ({ id, name: '', avatar: '' }))
  }
  const cacheData = service.cache.getCachedData()
  return userIds.map(userId => ({
    id: userId,
    name: cacheData.users[userId]?.name || '',
    avatar: cacheData.users[userId]?.avatar || qqAvatar(userId),
  }))
}

async function importRoleMembers(
  api: WebSocketAPIContext,
  params: { roleId: string, userIds: string[] },
  client: unknown,
) {
  try {
    const { roleId, userIds } = params
    if (!roleId || !Array.isArray(userIds)) {
      return error('无效的参数')
    }
    const scope = await api.resolveConsoleScope(client)
    requireAssignableRole(api.service.auth, scope, roleId)
    if (api.service.auth.isBuiltinRole(roleId)) {
      return error('内置角色由系统自动分配，不支持手动添加成员')
    }
    const imported = await assignImportedMembers(api.service, roleId, userIds)
    await api.service.data.authUsers.flush()
    return success({ success: true, imported })
  } catch (e) {
    return error(e instanceof Error ? e.message : '导入失败')
  }
}

async function assignImportedMembers(
  service: WebSocketAPIContext['service'],
  roleId: string,
  userIds: readonly string[],
) {
  let imported = 0
  for (const userId of userIds) {
    if (userId && typeof userId === 'string') {
      await service.auth.assignRole(userId.trim(), roleId)
      imported++
    }
  }
  return imported
}

async function listUsersByAuthority(
  api: WebSocketAPIContext,
  params: { authority: number },
  client: unknown,
) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'authority user query')
    if (typeof params.authority !== 'number' || params.authority < 1 || params.authority > 5) {
      return error('无效的权限等级')
    }
    const users = await api.ctx.database.get('user', { authority: params.authority })
    const bindings = await api.ctx.database.get('binding', { aid: { $in: users.map(u => u.id) } })
    return success(formatAuthorityUsers(api.service, users, bindings))
  } catch (e) {
    api.ctx.logger('stuhelperGroupCenter').error('获取权限用户列表失败:', e)
    return error(e instanceof Error ? e.message : '获取用户列表失败')
  }
}

function formatAuthorityUsers(service: WebSocketAPIContext['service'], users: any[], bindings: any[]) {
  const cacheData = service.cache.getCachedData()
  const aidToBinding = preferQQBindings(bindings)
  return users.filter(user => aidToBinding[user.id]).map(user => {
    const binding = aidToBinding[user.id]
    const cached = cacheData.users[binding.pid]
    return {
      id: binding.pid,
      name: cached?.name || user.name || '',
      avatar: cached?.avatar || (QQ_PLATFORMS.includes(binding.platform) ? qqAvatar(binding.pid) : ''),
    }
  })
}

function preferQQBindings(bindings: any[]) {
  const aidToBinding: Record<number, { platform: string; pid: string }> = {}
  for (const binding of bindings) {
    const existing = aidToBinding[binding.aid]
    if (!existing || (QQ_PLATFORMS.includes(binding.platform) && !QQ_PLATFORMS.includes(existing.platform))) {
      aidToBinding[binding.aid] = { platform: binding.platform, pid: binding.pid }
    }
  }
  return aidToBinding
}

async function listGuildAdmins(
  api: WebSocketAPIContext,
  params: { guildId: string },
  client: unknown,
) {
  try {
    if (!params.guildId) return error('缺少群号')
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, params.guildId, 'guild admin query')
    return await findGuildAdmins(api, params.guildId)
  } catch (e) {
    return error(e instanceof Error ? e.message : '获取群管理员列表失败')
  }
}

async function findGuildAdmins(api: WebSocketAPIContext, guildId: string) {
  for (const bot of api.ctx.bots) {
    try {
      return success(formatGuildAdmins(await fetchGuildMembers(bot, guildId), bot.platform))
    } catch (e) {
      api.ctx.logger('stuhelperGroupCenter').warn('获取群管理员列表失败:', e)
    }
  }
  return error('无法获取群管理员列表')
}

async function fetchGuildMembers(bot: any, guildId: string) {
  const members: any[] = []
  let next: string | undefined
  do {
    const result = await bot.getGuildMemberList(guildId, next)
    if (result.data) members.push(...result.data)
    next = result.next
  } while (next)
  return members
}

function formatGuildAdmins(members: any[], platform: string) {
  return members.filter(isGuildAdmin).map(member => {
    const userId = member.user?.id || member.userId
    const avatar = member.user?.avatar || member.avatar || (QQ_PLATFORMS.includes(platform) ? qqAvatar(userId) : '')
    return {
      id: userId,
      name: member.nick || member.user?.nick || member.user?.name || userId,
      avatar,
    }
  })
}

function isGuildAdmin(member: any) {
  const roles = member.roles || []
  const role = member.role
  return roles.includes('admin') || roles.includes('owner') || role === 'admin' || role === 'owner'
}

function qqAvatar(userId: string) {
  return `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
}
