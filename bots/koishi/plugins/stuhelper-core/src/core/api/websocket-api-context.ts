import type { Context } from 'koishi'

import type { StuhelperGroupCenterService } from '../services/stuhelper-group-center.service'
import type { Role, Subscription } from '../../types'
import { createAuthority4ListenerRegistrar } from './authority-listener'
import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  hasConsoleGuildAccess,
  resolveRequiredConsoleGuildScope,
} from './console-guild-scope'

export type AuthorityListenerRegistrar = ReturnType<typeof createAuthority4ListenerRegistrar>
export type ConsoleScopeResolver = ReturnType<typeof createConsoleScopeResolver>
export type ResolvedConsoleScope = Awaited<ReturnType<ConsoleScopeResolver>>

export interface WebSocketAPIContext {
  readonly ctx: Context
  readonly service: StuhelperGroupCenterService
  readonly data: StuhelperGroupCenterService['data']
  readonly addAuthorityListener: AuthorityListenerRegistrar
  readonly resolveConsoleScope: ConsoleScopeResolver
}

export function createWebSocketAPIContext(
  ctx: Context,
  service: StuhelperGroupCenterService,
): WebSocketAPIContext {
  if (!ctx.console) {
    throw new Error('console service is required to register WebSocket API')
  }
  return {
    ctx,
    service,
    data: service.data,
    addAuthorityListener: createAuthority4ListenerRegistrar(ctx.console),
    resolveConsoleScope: createConsoleScopeResolver(ctx, service),
  }
}

function createConsoleScopeResolver(ctx: Context, service: StuhelperGroupCenterService) {
  return (client: unknown) => resolveRequiredConsoleGuildScope(client as never, {
    roles: service.auth.getRoles(),
    getUserRoleIds: (userId: string) => service.auth.getUserRoleIds(userId),
    listBindingsByAuthId: (authId: number) => ctx.database.get('binding', { aid: authId }),
  })
}

export function filterGuildEntries<T>(
  entries: Record<string, T>,
  scope: ResolvedConsoleScope,
) {
  if (scope.kind === 'all') {
    return Object.entries(entries)
  }
  return Object.entries(entries).filter(([guildId]) => scope.guildIds.has(guildId))
}

export function filterRoles(roles: Role[], scope: ResolvedConsoleScope) {
  if (scope.kind === 'all') {
    return roles
  }
  return roles.filter((role) => role.guildIds?.some((guildId) => scope.guildIds.has(guildId)))
}

export function filterRoleIds(
  roleIds: readonly string[],
  roles: Role[],
  scope: ResolvedConsoleScope,
) {
  const visibleRoleIds = new Set(filterRoles(roles, scope).map((role) => role.id))
  return roleIds.filter((roleId) => visibleRoleIds.has(roleId))
}

export function assertReadableRole(
  roles: Role[],
  scope: ResolvedConsoleScope,
  roleId: string,
) {
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

export function filterSubscriptions(
  subscriptions: Subscription[],
  scope: ResolvedConsoleScope,
) {
  if (scope.kind === 'all') {
    return subscriptions
  }
  return subscriptions.filter((sub) => sub.type === 'group' && scope.guildIds.has(sub.id))
}

export function filterLogs(logs: any[], scope: ResolvedConsoleScope) {
  if (scope.kind === 'all') {
    return logs
  }
  return logs.filter((log) => scope.guildIds.has(String(log.guildId || '')))
}

export function parseWarnKey(key: string) {
  const [guildId, userId] = key.split(':')
  return guildId && userId ? { guildId, userId } : null
}

export function assertSubscriptionScope(scope: ResolvedConsoleScope, sub: Subscription) {
  if (sub.type === 'group') {
    assertConsoleGuildAccess(scope, sub.id, 'subscription')
    return
  }
  assertGlobalConsoleScope(scope, 'private subscription')
}

export function hasSubscriptionScope(scope: ResolvedConsoleScope, sub: Subscription) {
  if (sub.type === 'group') {
    return hasConsoleGuildAccess(scope, sub.id)
  }
  return scope.kind === 'all'
}

export function findScopedSubscriptionRawIndex(
  list: Subscription[],
  scope: ResolvedConsoleScope,
  scopedIndex: number,
) {
  if (!Number.isInteger(scopedIndex) || scopedIndex < 0) {
    return -1
  }
  let visibleIndex = 0
  for (let rawIndex = 0; rawIndex < list.length; rawIndex++) {
    if (!hasSubscriptionScope(scope, list[rawIndex])) {
      continue
    }
    if (visibleIndex === scopedIndex) {
      return rawIndex
    }
    visibleIndex++
  }
  return -1
}
