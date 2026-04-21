import assert from 'node:assert/strict'
import test from 'node:test'

import type { Role } from '../../types'

import {
  deliverChatMessageToClients,
  type ConsoleChatClient,
} from './chat-delivery'

test('deliverChatMessageToClients only sends guild messages to matching scoped admins', async () => {
  const sent: Array<{ clientId: string; type: string; body: Record<string, unknown> }> = []
  const clients = [
    createClient('global-admin', 1, 4, sent),
    createClient('guild-1001-admin', 2, 4, sent),
    createClient('guild-2002-admin', 3, 4, sent),
    createClient('viewer', 4, 3, sent),
  ]

  await deliverChatMessageToClients({
    clients,
    payload: {
      id: 'msg-1',
      guildId: '1001',
      content: 'hello',
    },
    roles: [
      createRole('role-1001', ['1001']),
      createRole('role-2002', ['2002']),
    ],
    getUserRoleIds: (userId) => ({
      'onebot:2001': ['role-1001'],
      'onebot:2002': ['role-2002'],
    }[userId] ?? []),
    listBindingsByAuthId: async (authId) => ({
      2: [{ platform: 'onebot', pid: '2001' }],
      3: [{ platform: 'onebot', pid: '2002' }],
    }[authId] ?? []),
  })

  assert.deepEqual(sent.map((item) => item.clientId), ['global-admin', 'guild-1001-admin'])
  assert.ok(sent.every((item) => item.type === 'stuhelperGroupCenter/chat/message'))
})

test('deliverChatMessageToClients only sends guildless messages to global admins', async () => {
  const sent: Array<{ clientId: string; type: string; body: Record<string, unknown> }> = []
  const clients = [
    createClient('global-admin', 1, 4, sent),
    createClient('guild-admin', 2, 4, sent),
  ]

  await deliverChatMessageToClients({
    clients,
    payload: {
      id: 'msg-2',
      content: 'system notice',
    },
    roles: [createRole('role-1001', ['1001'])],
    getUserRoleIds: (userId) => ({
      'onebot:2001': ['role-1001'],
    }[userId] ?? []),
    listBindingsByAuthId: async (authId) => ({
      2: [{ platform: 'onebot', pid: '2001' }],
    }[authId] ?? []),
  })

  assert.deepEqual(sent.map((item) => item.clientId), ['global-admin'])
})

test('deliverChatMessageToClients treats empty or missing guildIds as global visibility', async () => {
  const sent: Array<{ clientId: string; type: string; body: Record<string, unknown> }> = []
  const clients = [createClient('global-role-admin', 2, 4, sent)]

  await deliverChatMessageToClients({
    clients,
    payload: {
      id: 'msg-3',
      guildId: '2002',
      content: 'ops notice',
    },
    roles: [
      createRole('role-global-empty-list', []),
      createRole('role-global-missing'),
    ],
    getUserRoleIds: (userId) => ({
      'onebot:2001': ['role-global-empty-list', 'role-global-missing'],
    }[userId] ?? []),
    listBindingsByAuthId: async () => [{ platform: 'onebot', pid: '2001' }],
  })

  assert.deepEqual(sent.map((item) => item.clientId), ['global-role-admin'])
})

function createClient(
  id: string,
  authId: number,
  authority: number,
  sent: Array<{ clientId: string; type: string; body: Record<string, unknown> }>,
): ConsoleChatClient {
  return {
    id,
    auth: {
      id: authId,
      authority,
    },
    send(payload) {
      sent.push({
        clientId: id,
        type: String(payload.type),
        body: payload.body as Record<string, unknown>,
      })
    },
  }
}

function createRole(id: string, guildIds?: string[]): Pick<Role, 'id' | 'guildIds'> {
  return { id, guildIds }
}
