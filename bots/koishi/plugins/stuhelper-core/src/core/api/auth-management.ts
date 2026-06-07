import type { Role } from '../../types'

import {
  assertConsoleGuildAccess,
  type ConsoleGuildScope,
} from './console-guild-scope'

interface RoleCatalog {
  isBuiltinRole: (roleId: string) => boolean
  getRoles: () => Role[]
}

interface RoleReorderInput {
  readonly sourceRoleId: string
  readonly targetRoleId: string
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

export function buildManagedRoleReorder(
  catalog: RoleCatalog,
  scope: ConsoleGuildScope,
  input: RoleReorderInput,
): [Role, Role] {
  const sourceRoleId = normalizeRoleId(input.sourceRoleId, 'source role')
  const targetRoleId = normalizeRoleId(input.targetRoleId, 'target role')
  if (sourceRoleId === targetRoleId) {
    throw new Error('source and target roles must be different')
  }

  const sourceRole = requireAssignableRole(catalog, scope, sourceRoleId)
  const targetRole = requireAssignableRole(catalog, scope, targetRoleId)
  return [
    { ...sourceRole, priority: targetRole.priority },
    { ...targetRole, priority: sourceRole.priority },
  ]
}

function normalizeRoleId(value: string, label: string) {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`invalid ${label}`)
  }
  return value.trim()
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
