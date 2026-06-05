import type { Binding } from 'koishi'

import type { Role } from '../../types'

const DEFAULT_MIN_AUTHORITY = 4

export interface ScopedConsoleClient {
  auth?: {
    id: number
    authority: number
  }
}

export interface ConsoleGuildScopeDeps {
  roles: readonly Pick<Role, 'id' | 'guildIds'>[]
  getUserRoleIds: (userId: string) => readonly string[]
  listBindingsByAuthId: (authId: number) => Promise<ReadonlyArray<Pick<Binding, 'platform' | 'pid'>>>
  minAuthority?: number
}

export type ConsoleGuildScope =
  | { kind: 'all' }
  | { kind: 'guilds'; guildIds: Set<string> }

export async function resolveConsoleGuildScope(
  client: unknown,
  deps: ConsoleGuildScopeDeps,
): Promise<ConsoleGuildScope | null> {
  const auth = readConsoleAuth(client)
  if (!auth || auth.authority < (deps.minAuthority ?? DEFAULT_MIN_AUTHORITY)) {
    return null
  }

  const authId = auth.id
  const bindings = await deps.listBindingsByAuthId(authId)
  const roleIds = new Set<string>()

  collectRoleIds(roleIds, deps, String(authId))
  for (const binding of bindings) {
    collectRoleIds(roleIds, deps, `${binding.platform}:${binding.pid}`)
    collectRoleIds(roleIds, deps, binding.pid)
  }

  if (roleIds.size === 0) {
    return { kind: 'all' }
  }

  const rolesById = new Map(deps.roles.map((role) => [role.id, role]))
  const guildIds = new Set<string>()
  for (const roleId of roleIds) {
    const role = rolesById.get(roleId)
    if (!role) {
      throw new Error(`console role assignment references missing role: ${roleId}`)
    }
    if (!role.guildIds || role.guildIds.length === 0) {
      return { kind: 'all' }
    }
    role.guildIds.forEach((guildId) => guildIds.add(guildId))
  }

  if (guildIds.size === 0) {
    return { kind: 'all' }
  }

  return {
    kind: 'guilds',
    guildIds,
  }
}

export async function resolveRequiredConsoleGuildScope(
  client: unknown,
  deps: ConsoleGuildScopeDeps,
) {
  const auth = readConsoleAuth(client)
  if (!auth) {
    throw new Error('console auth is required')
  }

  const scope = await resolveConsoleGuildScope(client, deps)
  if (!scope) {
    throw new Error(`console authority must be >= ${deps.minAuthority ?? DEFAULT_MIN_AUTHORITY}`)
  }
  return scope
}

export function hasConsoleGuildAccess(
  scope: ConsoleGuildScope,
  guildId: string | undefined,
) {
  if (scope.kind === 'all') {
    return true
  }
  if (!guildId) {
    return false
  }
  return scope.guildIds.has(guildId)
}

export function assertConsoleGuildAccess(
  scope: ConsoleGuildScope,
  guildId: string | undefined,
  resource: string,
) {
  if (hasConsoleGuildAccess(scope, guildId)) {
    return
  }
  throw new Error(`${resource} is outside of the current console guild scope: ${guildId || '(missing guildId)'}`)
}

export function assertGlobalConsoleScope(
  scope: ConsoleGuildScope,
  resource: string,
) {
  if (scope.kind === 'all') {
    return
  }
  throw new Error(`${resource} requires global console scope`)
}

function collectRoleIds(
  target: Set<string>,
  deps: Pick<ConsoleGuildScopeDeps, 'getUserRoleIds'>,
  userId: string,
) {
  deps.getUserRoleIds(userId).forEach((roleId) => target.add(roleId))
}

function readConsoleAuth(client: unknown): ScopedConsoleClient['auth'] | null {
  if (!isRecord(client) || !isRecord(client.auth)) return null
  const { id, authority } = client.auth
  if (typeof id !== 'number' || typeof authority !== 'number') return null
  return { id, authority }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
