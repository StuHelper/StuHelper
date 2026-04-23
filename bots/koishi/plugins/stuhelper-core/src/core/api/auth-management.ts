import type { Role } from '../../types'

import {
  assertConsoleGuildAccess,
  type ConsoleGuildScope,
} from './console-guild-scope'

interface RoleCatalog {
  isBuiltinRole: (roleId: string) => boolean
  getRoles: () => Role[]
}

export function normalizeManagedRoleInput(
  catalog: RoleCatalog,
  scope: ConsoleGuildScope,
  role: Role,
) {
  if (catalog.isBuiltinRole(role.id) || role.builtin) {
    throw new Error(`builtin role cannot be modified: ${role.id}`)
  }

  const guildIds = normalizeGuildIds(role.guildIds)
  if (scope.kind === 'guilds') {
    if (guildIds.length === 0) {
      throw new Error('global roles require global console scope')
    }
    guildIds.forEach((guildId) => assertConsoleGuildAccess(scope, guildId, 'role guild'))
  }

  return {
    ...role,
    guildIds,
  }
}

export function requireAssignableRole(
  catalog: RoleCatalog,
  scope: ConsoleGuildScope,
  roleId: string,
) {
  const role = catalog.getRoles().find((item) => item.id === roleId)
  if (!role) {
    throw new Error(`role not found: ${roleId}`)
  }
  if (catalog.isBuiltinRole(role.id) || role.builtin) {
    throw new Error(`builtin role cannot be modified: ${role.id}`)
  }

  const guildIds = normalizeGuildIds(role.guildIds)
  if (scope.kind === 'guilds') {
    if (guildIds.length === 0) {
      throw new Error('global roles require global console scope')
    }
    guildIds.forEach((guildId) => assertConsoleGuildAccess(scope, guildId, 'role guild'))
  }

  return role
}

function normalizeGuildIds(guildIds: readonly string[] | undefined) {
  if (!guildIds || guildIds.length === 0) {
    return []
  }
  return Array.from(new Set(
    guildIds
      .map((guildId) => guildId.trim())
      .filter(Boolean),
  ))
}
