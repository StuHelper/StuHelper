import assert from 'node:assert/strict'
import test from 'node:test'

import { registerMemberBlacklistConsoleAPI } from './member-blacklist-console-api'

type Listener = (params: any) => Promise<{ success: boolean; data?: any }>

test('console blacklist add includes manual admin metadata required by platform', async () => {
  const listeners = new Map<string, Listener>()
  const createdBlacklists: Array<Record<string, any>> = []
  const releasedById: Array<{ id: string; request: Record<string, any> }> = []

  registerMemberBlacklistConsoleAPI(
    createConsoleContext(listeners) as any,
    createConsoleService() as any,
    createBlacklistBackend(createdBlacklists, releasedById) as any,
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
  assert.equal(createdBlacklists[0].createdFrom, 'koishi_console')
  assert.equal(createdBlacklists[0].metadata.operatorInput, '10001')
  assert.equal(createdBlacklists[0].metadata.scopeSelectionContext, 'koishi_console_form')
  assert.equal(createdBlacklists[0].metadata.createdFrom, undefined)
})

test('console blacklist remove forwards id-based release with chosen reason code', async () => {
  const listeners = new Map<string, Listener>()
  const createdBlacklists: Array<Record<string, any>> = []
  const releasedById: Array<{ id: string; request: Record<string, any> }> = []

  registerMemberBlacklistConsoleAPI(
    createConsoleContext(listeners) as any,
    createConsoleService() as any,
    createBlacklistBackend(createdBlacklists, releasedById, async () => ({
      list: [memberBlacklistEntry('entry-42', { guildID: '1001' })],
      total: 1,
    })) as any,
  )

  const result = await callListener(listeners, 'stuhelperGroupCenter/blacklist/remove', {
    id: 'entry-42',
    scopeType: 'guild',
    guildID: '1001',
    releaseReasonCode: 'release_only',
  })

  assert.equal(result.success, true)
  assert.equal(releasedById.length, 1)
  assert.equal(releasedById[0].id, 'entry-42')
  assert.equal(releasedById[0].request.releaseReasonCode, 'release_only')
  assert.equal(releasedById[0].request.operatorQQID, undefined)
})

test('console blacklist remove checks actual visible entry scope before release', async () => {
  const listeners = new Map<string, Listener>()
  const createdBlacklists: Array<Record<string, any>> = []
  const releasedById: Array<{ id: string; request: Record<string, any> }> = []

  registerMemberBlacklistConsoleAPI(
    createConsoleContext(listeners) as any,
    createConsoleService({ guildIds: ['1001'] }) as any,
    createBlacklistBackend(createdBlacklists, releasedById, async () => ({
      list: [memberBlacklistEntry('entry-42', { scopeType: 'global', guildID: null })],
      total: 1,
    })) as any,
  )

  await assert.rejects(
    callListener(listeners, 'stuhelperGroupCenter/blacklist/remove', {
      id: 'entry-42',
      scopeType: 'guild',
      guildID: '1001',
      releaseReasonCode: 'manual_pardon',
    }),
    /global member blacklist/,
  )
  assert.equal(releasedById.length, 0)
})

test('console blacklist list fetches all backend pages', async () => {
  const listeners = new Map<string, Listener>()
  const listCalls: Array<Record<string, any>> = []

  registerMemberBlacklistConsoleAPI(
    createConsoleContext(listeners) as any,
    createConsoleService() as any,
    createBlacklistBackend([], [], async (input) => {
      listCalls.push(input)
      const page = input.page || 1
      return {
        list: [memberBlacklistEntry(`entry-${page}`)],
        total: 2,
      }
    }) as any,
  )

  const result = await callListener(listeners, 'stuhelperGroupCenter/blacklist/list', {})

  assert.equal(result.success, true)
  assert.equal(result.data.total, 2)
  assert.deepEqual(result.data.list.map((entry: any) => entry.id), ['entry-1', 'entry-2'])
  assert.deepEqual(listCalls.map((call) => call.page), [1, 2])
  assert.ok(listCalls.every((call) => call.platform === 'qq'))
})

test('console blacklist remove rejects unsupported release reason code', async () => {
  const listeners = new Map<string, Listener>()
  const createdBlacklists: Array<Record<string, any>> = []
  const releasedById: Array<{ id: string; request: Record<string, any> }> = []

  registerMemberBlacklistConsoleAPI(
    createConsoleContext(listeners) as any,
    createConsoleService() as any,
    createBlacklistBackend(createdBlacklists, releasedById) as any,
  )

  await assert.rejects(
    callListener(listeners, 'stuhelperGroupCenter/blacklist/remove', {
      id: 'entry-42',
      scopeType: 'guild',
      guildID: '1001',
      releaseReasonCode: 'policy_expired_auto',
    }),
    /unsupported releaseReasonCode/,
  )
  assert.equal(releasedById.length, 0)
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

function createConsoleService(options: { guildIds?: string[] } = {}) {
  const guildIds = options.guildIds || []
  return {
    auth: {
      getRoles: () => [{ id: 'role-1', guildIds }],
      getUserRoleIds: () => ['role-1'],
    },
  }
}

function createBlacklistBackend(
  createdBlacklists: Array<Record<string, any>>,
  releasedById: Array<{ id: string; request: Record<string, any> }>,
  listMemberBlacklist: (input: Record<string, any>) => Promise<{ list: any[]; total: number }> =
    async () => ({ list: [], total: 0 }),
) {
  return {
    async createMemberBlacklist(input: Record<string, any>) {
      createdBlacklists.push(input)
      return { id: 'entry-1', ...input }
    },
    listMemberBlacklist,
    async releaseMemberBlacklist(id: string, request: Record<string, any>) {
      releasedById.push({ id, request })
      return { id }
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

function memberBlacklistEntry(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: '1001',
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: 'manual',
    metadata: {},
    createdByType: 'service_account',
    createdByID: 'koishi-runtime',
    createdFrom: 'koishi_console',
    createdAt: '2026-05-08T00:00:00Z',
    updatedAt: '2026-05-08T00:00:00Z',
    ...overrides,
  }
}
