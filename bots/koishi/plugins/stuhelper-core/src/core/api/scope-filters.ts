import type { Role, Subscription } from '../../types'
import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  hasConsoleGuildAccess,
  type ConsoleGuildScope,
} from './console-guild-scope'

export function filterGuildEntries<T>(entries: Record<string, T>, scope: ConsoleGuildScope) {
  if (scope.kind === 'all') {
    return Object.entries(entries)
  }
  return Object.entries(entries).filter(([guildId]) => scope.guildIds.has(guildId))
}

export function filterRoles(roles: Role[], scope: ConsoleGuildScope) {
  if (scope.kind === 'all') {
    return roles
  }
  return roles.filter((role) => role.guildIds?.some((guildId) => scope.guildIds.has(guildId)))
}

export function filterRoleIds(roleIds: readonly string[], roles: Role[], scope: ConsoleGuildScope) {
  const visibleRoleIds = new Set(filterRoles(roles, scope).map((role) => role.id))
  return roleIds.filter((roleId) => visibleRoleIds.has(roleId))
}

export function filterSubscriptions(subscriptions: Subscription[], scope: ConsoleGuildScope) {
  if (scope.kind === 'all') {
    return subscriptions
  }
  return subscriptions.filter((sub) => sub.type === 'group' && scope.guildIds.has(sub.id))
}

export function filterLogs<T extends { guildId?: unknown }>(logs: T[], scope: ConsoleGuildScope): T[] {
  if (scope.kind === 'all') {
    return logs
  }
  return logs.filter((log) => scope.guildIds.has(String(log.guildId || '')))
}

export function parseWarnKey(key: string) {
  const [guildId, userId] = key.split(':')
  return guildId && userId ? { guildId, userId } : null
}

export function assertReadableRole(scope: ConsoleGuildScope, roles: Role[], roleId: string) {
  const role = roles.find((item) => item.id === roleId)
  if (!role) {
    throw new Error(`role not found: ${roleId}`)
  }
  if (scope.kind === 'all') {
    return
  }
  if (!role.guildIds?.length) {
    assertGlobalConsoleScope(scope, 'global role')
    return
  }
  if (role.guildIds.some((guildId) => scope.guildIds.has(guildId))) {
    return
  }
  assertConsoleGuildAccess(scope, role.guildIds[0], 'role')
}

export function assertSubscriptionScope(scope: ConsoleGuildScope, sub: Subscription) {
  if (sub.type === 'group') {
    assertConsoleGuildAccess(scope, sub.id, 'subscription')
    return
  }
  assertGlobalConsoleScope(scope, 'private subscription')
}

export function hasSubscriptionScope(scope: ConsoleGuildScope, sub: Subscription) {
  if (sub.type === 'group') {
    return hasConsoleGuildAccess(scope, sub.id)
  }
  return scope.kind === 'all'
}

export function findScopedSubscriptionRawIndex(
  list: Subscription[],
  scope: ConsoleGuildScope,
  scopedIndex: number,
) {
  if (!Number.isInteger(scopedIndex) || scopedIndex < 0) {
    return -1
  }

  let visibleIndex = 0
  for (let rawIndex = 0; rawIndex < list.length; rawIndex++) {
    if (!hasSubscriptionScope(scope, list[rawIndex])) continue
    if (visibleIndex === scopedIndex) return rawIndex
    visibleIndex++
  }
  return -1
}
