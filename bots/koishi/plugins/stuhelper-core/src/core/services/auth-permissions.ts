import type { Session } from 'koishi'

import type { Role } from '../../types'
import { isGuildAdminMember } from './auth-guild-admin'

interface AuthorityHolder {
  readonly authority?: unknown
}

export function collectSessionPermissions(input: {
  readonly session: Session
  readonly roles: Record<string, Role>
  readonly userRoleIds: readonly string[]
}): Set<string> {
  const perms = new Set<string>()
  const guildId = input.session.guildId
  const authority = getSessionAuthority(input.session)

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
  return isGuildAdminMember(session.author)
}

function getSessionAuthority(session: Session): number {
  const authority = isAuthorityHolder(session.user) ? session.user.authority : 0
  return typeof authority === 'number' ? authority : 0
}

function isAuthorityHolder(value: unknown): value is AuthorityHolder {
  return typeof value === 'object' && value !== null && 'authority' in value
}
