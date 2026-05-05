import assert from 'node:assert/strict'
import test from 'node:test'

import { registerMemberBlacklistConsoleAPI } from './member-blacklist-console-api'

type Listener = (params: any) => Promise<{ success: boolean; data?: any }>

test('console blacklist add includes manual admin metadata required by platform', async () => {
  const listeners = new Map<string, Listener>()
  const createdBlacklists: Array<Record<string, any>> = []

  registerMemberBlacklistConsoleAPI(
    createConsoleContext(listeners) as any,
    createConsoleService() as any,
    createBlacklistBackend(createdBlacklists) as any,
  )

  const result = await callListener(listeners, 'stuhelperGroupCenter/blacklist/add', {
    platform: 'qq',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: '1001',
    reasonText: 'manual note',
  })

  assert.equal(result.success, true)
  assert.equal(createdBlacklists.length, 1)
  assert.equal(createdBlacklists[0].metadata.operatorInput, '10001')
  assert.equal(createdBlacklists[0].metadata.scopeSelectionContext, 'koishi_console_form')
})

async function callListener(
  listeners: Map<string, Listener>,
  event: string,
  params: Record<string, unknown>,
) {
  const listener = listeners.get(event)
  assert.ok(listener, `${event} listener should be registered`)
  return listener.call(createConsoleClient(), params)
}

function createConsoleContext(listeners: Map<string, Listener>) {
  return {
    console: {
      addListener(event: string, callback: Listener) {
        listeners.set(event, callback)
      },
    },
    database: {
      get: async () => [{ aid: 42, platform: 'qq', pid: 'operator' }],
    },
  }
}

function createConsoleService() {
  return {
    auth: {
      getRoles: () => [{ id: 'global-role', guildIds: [] }],
      getUserRoleIds: () => ['global-role'],
    },
  }
}

function createBlacklistBackend(createdBlacklists: Array<Record<string, any>>) {
  return {
    async createMemberBlacklist(input: Record<string, any>) {
      createdBlacklists.push(input)
      return { id: 'entry-1', ...input }
    },
    async listMemberBlacklist() {
      return { list: [], total: 0 }
    },
    async releaseMemberBlacklistBySubject() {
      return { id: 'entry-1' }
    },
  }
}

function createConsoleClient() {
  return {
    auth: {
      id: 42,
      authority: 4,
    },
  }
}
