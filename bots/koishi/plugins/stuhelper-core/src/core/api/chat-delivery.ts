import type { Binding } from 'koishi'

import type { Role } from '../../types'

const CHAT_EVENT_TYPE = 'stuhelperGroupCenter/chat/message'
const DEFAULT_MIN_AUTHORITY = 4

export interface ConsoleChatPayload {
  guildId?: string
  [key: string]: unknown
}

export interface ConsoleChatClient {
  id: string
  auth?: {
    id: number
    authority: number
  }
  send(payload: { type: string; body: ConsoleChatPayload }): void
}

interface ChatGuildScope {
  kind: 'all' | 'guilds'
  guildIds?: Set<string>
}

interface ChatDeliveryDeps {
  clients: readonly ConsoleChatClient[]
  payload: ConsoleChatPayload
  roles: readonly Pick<Role, 'id' | 'guildIds'>[]
  getUserRoleIds: (userId: string) => readonly string[]
  listBindingsByAuthId: (authId: number) => Promise<ReadonlyArray<Pick<Binding, 'platform' | 'pid'>>>
  minAuthority?: number
}

export async function deliverChatMessageToClients(deps: ChatDeliveryDeps) {
  const deliveredClientIds: string[] = []
  const rolesById = new Map(deps.roles.map((role) => [role.id, role]))
  const scopeCache = new Map<number, Promise<ChatGuildScope | null>>()

  for (const client of deps.clients) {
    const scope = await resolveClientScope(client, deps, rolesById, scopeCache)
    if (!scope || !shouldDeliverPayload(scope, deps.payload.guildId)) {
      continue
    }

    client.send({
      type: CHAT_EVENT_TYPE,
      body: deps.payload,
    })
    deliveredClientIds.push(client.id)
  }

  return deliveredClientIds
}

async function resolveClientScope(
  client: ConsoleChatClient,
  deps: ChatDeliveryDeps,
  rolesById: ReadonlyMap<string, Pick<Role, 'id' | 'guildIds'>>,
  scopeCache: Map<number, Promise<ChatGuildScope | null>>,
) {
  if (!client.auth || client.auth.authority < (deps.minAuthority ?? DEFAULT_MIN_AUTHORITY)) {
    return null
  }

  const authId = client.auth.id
  const cached = scopeCache.get(authId)
  if (cached) {
    return cached
  }

  const pending = buildClientScope(authId, deps, rolesById)
  scopeCache.set(authId, pending)
  return pending
}

async function buildClientScope(
  authId: number,
  deps: ChatDeliveryDeps,
  rolesById: ReadonlyMap<string, Pick<Role, 'id' | 'guildIds'>>,
) {
  const bindings = await deps.listBindingsByAuthId(authId)
  const roleIds = new Set<string>()

  roleIdsFromUser(deps, String(authId)).forEach((roleId) => roleIds.add(roleId))
  for (const binding of bindings) {
    roleIdsFromUser(deps, `${binding.platform}:${binding.pid}`).forEach((roleId) => roleIds.add(roleId))
    roleIdsFromUser(deps, binding.pid).forEach((roleId) => roleIds.add(roleId))
  }

  if (roleIds.size === 0) {
    return { kind: 'all' } as const
  }

  const scopedGuildIds = new Set<string>()
  for (const roleId of roleIds) {
    const role = rolesById.get(roleId)
    if (!role) {
      throw new Error(`console role assignment references missing role: ${roleId}`)
    }
    if (!role.guildIds || role.guildIds.length === 0) {
      return { kind: 'all' } as const
    }
    role.guildIds.forEach((guildId) => scopedGuildIds.add(guildId))
  }

  if (scopedGuildIds.size === 0) {
    return { kind: 'all' } as const
  }

  return {
    kind: 'guilds',
    guildIds: scopedGuildIds,
  } as const
}

function roleIdsFromUser(deps: ChatDeliveryDeps, userId: string) {
  return deps.getUserRoleIds(userId)
}

function shouldDeliverPayload(scope: ChatGuildScope, guildId: string | undefined) {
  if (scope.kind === 'all') {
    return true
  }
  if (!guildId) {
    return false
  }
  return scope.guildIds?.has(guildId) ?? false
}
