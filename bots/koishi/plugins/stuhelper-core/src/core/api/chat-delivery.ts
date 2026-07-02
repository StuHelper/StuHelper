import type { Binding } from 'koishi'

import type { Role } from '../../types'

import {
  type ConsoleGuildScope,
  type ConsoleGuildScopeDeps,
  hasConsoleGuildAccess,
  resolveConsoleGuildScope,
} from './console-guild-scope'

const CHAT_EVENT_TYPE = 'stuhelperGroupCenter/chat/message'

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
  const scopeDeps: ConsoleGuildScopeDeps = {
    roles: deps.roles,
    getUserRoleIds: deps.getUserRoleIds,
    listBindingsByAuthId: deps.listBindingsByAuthId,
    minAuthority: deps.minAuthority,
  }
  // 同一次广播里多个 console 连接可能属于同一账号，按 authId 复用范围解析结果。
  const scopeCache = new Map<number, Promise<ConsoleGuildScope | null>>()

  for (const client of deps.clients) {
    const scope = await resolveCachedScope(client, scopeDeps, scopeCache)
    if (!scope || !hasConsoleGuildAccess(scope, deps.payload.guildId)) {
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

function resolveCachedScope(
  client: ConsoleChatClient,
  deps: ConsoleGuildScopeDeps,
  cache: Map<number, Promise<ConsoleGuildScope | null>>,
): Promise<ConsoleGuildScope | null> {
  if (!client.auth) {
    return Promise.resolve(null)
  }
  const cached = cache.get(client.auth.id)
  if (cached) {
    return cached
  }
  const pending = resolveConsoleGuildScope(client, deps)
  cache.set(client.auth.id, pending)
  return pending
}
