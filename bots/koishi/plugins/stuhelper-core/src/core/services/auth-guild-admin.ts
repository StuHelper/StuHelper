import type { GuildMember, GuildRole } from '@satorijs/protocol'

const GUILD_ADMIN_ROLE_IDS = new Set(['admin', 'owner'])

export interface LegacyGuildRoleHolder {
  readonly role?: unknown
}

export type GuildAdminMember = GuildMember & LegacyGuildRoleHolder

export function isGuildAdminMember(member: GuildAdminMember | null | undefined): boolean {
  if (!member) return false

  const roles = member.roles || []
  if (roles.some(isGuildAdminRole)) return true
  return isGuildAdminRole(member.role)
}

export function isGuildAdminRole(role: unknown): boolean {
  if (typeof role === 'string') return GUILD_ADMIN_ROLE_IDS.has(role)
  if (!isGuildRoleLike(role)) return false
  return GUILD_ADMIN_ROLE_IDS.has(role.id) || Boolean(role.name && GUILD_ADMIN_ROLE_IDS.has(role.name))
}

function isGuildRoleLike(role: unknown): role is Pick<GuildRole, 'id' | 'name'> {
  return typeof role === 'object' && role !== null && typeof (role as { readonly id?: unknown }).id === 'string'
}
