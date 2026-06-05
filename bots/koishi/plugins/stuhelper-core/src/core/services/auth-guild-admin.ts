import type { GuildMember, GuildRole } from '@satorijs/protocol'

const GUILD_ADMIN_ROLE_IDS = new Set(['admin', 'owner'])
const GUILD_OWNER_ROLE_IDS = new Set(['owner'])

export interface LegacyGuildRoleHolder {
  readonly role?: unknown
}

export type GuildAdminMember = GuildMember & LegacyGuildRoleHolder

export function isGuildAdminMember(member: GuildAdminMember | null | undefined): boolean {
  return hasGuildRole(member, GUILD_ADMIN_ROLE_IDS)
}

export function isGuildOwnerMember(member: GuildAdminMember | null | undefined): boolean {
  return hasGuildRole(member, GUILD_OWNER_ROLE_IDS)
}

function hasGuildRole(member: GuildAdminMember | null | undefined, roleIds: ReadonlySet<string>): boolean {
  if (!member) return false

  const roles = member.roles || []
  if (roles.some((role) => isGuildRole(role, roleIds))) return true
  return isGuildRole(member.role, roleIds)
}

export function isGuildAdminRole(role: unknown): boolean {
  return isGuildRole(role, GUILD_ADMIN_ROLE_IDS)
}

function isGuildRole(role: unknown, roleIds: ReadonlySet<string>): boolean {
  if (typeof role === 'string') return roleIds.has(role)
  if (!isGuildRoleLike(role)) return false
  return roleIds.has(role.id) || Boolean(role.name && roleIds.has(role.name))
}

function isGuildRoleLike(role: unknown): role is Pick<GuildRole, 'id' | 'name'> {
  return typeof role === 'object' && role !== null && typeof (role as { readonly id?: unknown }).id === 'string'
}
