import type { Session } from 'koishi'

import type { Role } from '../../types'

export function collectSessionPermissions(input: {
  readonly session: Session
  readonly roles: Record<string, Role>
  readonly userRoleIds: readonly string[]
}): Set<string> {
  const perms = new Set<string>()
  const guildId = input.session.guildId
  const authority = (input.session.user as any)?.authority || 0

  addAuthorityRolePermissions(perms, input.roles, authority)
  if (isGuildAdmin(input.session) && guildId) {
    addRolePermissions({ perms, roles: input.roles, roleId: 'guild-admin', options: { guildId, checkGuildScope: false } })
  }
  input.userRoleIds.forEach((roleId) => {
    addRolePermissions({ perms, roles: input.roles, roleId, options: { guildId, checkGuildScope: true } })
  })
  return perms
}

export function hasPermissionNode(perms: ReadonlySet<string>, node: string): boolean {
  if (perms.has(node) || perms.has('*')) return true

  const parts = node.split('.')
  let current = ''
  for (let i = 0; i < parts.length - 1; i++) {
    current += (i === 0 ? '' : '.') + parts[i]
    if (perms.has(`${current}.*`)) return true
  }
  return false
}

function addAuthorityRolePermissions(
  perms: Set<string>,
  roles: Record<string, Role>,
  authority: number,
) {
  if (authority >= 1) addRolePermissions({ perms, roles, roleId: 'authority1', options: { checkGuildScope: false } })
  if (authority >= 2) addRolePermissions({ perms, roles, roleId: 'authority2', options: { checkGuildScope: false } })
  if (authority >= 3) addRolePermissions({ perms, roles, roleId: 'authority3', options: { checkGuildScope: false } })
  if (authority >= 4) addRolePermissions({ perms, roles, roleId: 'authority4+', options: { checkGuildScope: false } })
}

function addRolePermissions(input: {
  readonly perms: Set<string>
  readonly roles: Record<string, Role>
  readonly roleId: string
  readonly options: { guildId?: string, checkGuildScope: boolean }
}) {
  const { perms, roles, roleId, options } = input
  const role = roles[roleId]
  if (!role || !isRoleVisibleInGuild(role, options)) return
  role.permissions?.forEach((permission) => perms.add(permission))
}

function isRoleVisibleInGuild(role: Role, options: { guildId?: string, checkGuildScope: boolean }) {
  if (!options.checkGuildScope) return true
  const roleGuildIds = role.guildIds || []
  return roleGuildIds.length === 0 || Boolean(options.guildId && roleGuildIds.includes(options.guildId))
}

function isGuildAdmin(session: Session): boolean {
  const author = session.author || (session.event as any)?.member
  if (!author) return false

  const roles = author.roles || []
  if (roles.includes('admin') || roles.includes('owner')) return true
  const role = (author as any).role
  return role === 'admin' || role === 'owner'
}
