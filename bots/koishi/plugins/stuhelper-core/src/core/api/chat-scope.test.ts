import assert from 'node:assert/strict'
import test from 'node:test'

import { registerWebSocketAPI } from './index'

type Listener = (params: any) => Promise<{ success: boolean; error?: string }>

test('chat guild APIs reject guilds outside the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/chat/guild-members', { guildId: '2002' })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/chat/guild-info', { guildId: '2002' })
})

test('config and role read APIs filter data to the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const configs = await callListener(listeners, 'stuhelperGroupCenter/config/list', {})
  assert.deepEqual(Object.keys(configs.data || {}), ['1001'])

  const config = await callListener(listeners, 'stuhelperGroupCenter/config/get', { guildId: '2002' })
  assert.equal(config.success, false)
  assert.match(config.error || '', /outside of the current console guild scope/)

  const roles = await callListener(listeners, 'stuhelperGroupCenter/auth/role/list', {})
  assert.deepEqual((roles.data || []).map((role: { id: string }) => role.id), ['scoped-role'])
})

test('chat write APIs reject messages outside the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/chat/send', {
    channelId: '2002',
    content: 'hello',
    guildId: '2002',
  })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/chat/recall', {
    channelId: '2002',
    guildId: '2002',
    messageId: 'msg-1',
  })
})

test('chat send rejects oversized content', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const result = await callListener(listeners, 'stuhelperGroupCenter/chat/send', {
    channelId: '1001',
    content: 'x'.repeat(262145),
    guildId: '1001',
  })

  assert.equal(result.success, false)
  assert.match(result.error || '', /message content is too large/)
})

test('legacy warn APIs filter and reject data outside the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const warns = await callListener(listeners, 'stuhelperGroupCenter/warns/list', {})
  assert.deepEqual((warns.data || []).map((item: { key: string }) => item.key), ['1001:u1'])

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/warns/get', { key: '2002:u2' })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/warns/add', { guildId: '2002', userId: 'u2' })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/warns/update', { key: '2002:u2', count: 2 } as any)
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/warns/clear', { key: '2002:u2' })
})

test('legacy subscription and cache APIs enforce the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const subscriptions = await callListener(listeners, 'stuhelperGroupCenter/subscriptions/list', {})
  assert.deepEqual((subscriptions.data || []).map((item: { id: string }) => item.id), ['1001'])

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/subscriptions/add', {
    subscription: { type: 'group', id: '2002', features: {} },
  } as any)
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/cache/fetch-name', {
    type: 'guild',
    guildId: '2002',
  })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/cache/fetch-name', {
    type: 'member',
    guildId: '2002',
    userId: 'u2',
  })
})

test('subscription remove uses the visible scoped subscription identity instead of raw list index', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001', '1003'])
  service.data.subscriptions.set('list', [
    { type: 'group', id: '2002', features: { hidden: true } },
    { type: 'group', id: '1001', features: { visible: 1 } },
    { type: 'group', id: '1003', features: { visible: 2 } },
  ] as any)

  registerWebSocketAPI(ctx as any, service as any)

  const subscriptions = await callListener(listeners, 'stuhelperGroupCenter/subscriptions/list', {})
  assert.deepEqual((subscriptions.data || []).map((item: { id: string }) => item.id), ['1001', '1003'])

  const result = await callListener(listeners, 'stuhelperGroupCenter/subscriptions/remove', { index: 1 })

  assert.equal(result.success, true)
  assert.deepEqual(
    service.data.subscriptions.get('list').map((item: { id: string }) => item.id),
    ['2002', '1001'],
  )
})

test('legacy global APIs reject guild-scoped console users', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/cache/clear', {})
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/cache/refresh', {})
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/cache/stats', {})
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/chat/user-info', { userId: 'u2' })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/subscriptions/update', {
    index: 0,
    subscription: { type: 'group', id: '2002', features: {} },
  } as any)
  await assertRejectsNotFound(listeners, 'stuhelperGroupCenter/subscriptions/remove', { index: 1 })
})

test('legacy stats APIs filter guild data to the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const dashboard = await callListener(listeners, 'stuhelperGroupCenter/stats/dashboard', {})
  assert.equal(dashboard.data.totalGroups, 1)
  assert.equal(dashboard.data.totalWarns, 1)
  assert.equal(dashboard.data.totalBlacklisted, 1)
  assert.equal(dashboard.data.totalSubscriptions, 1)

  const charts = await callListener(listeners, 'stuhelperGroupCenter/stats/charts', {})
  assert.deepEqual(charts.data.guildRank.map((item: { guildId: string }) => item.guildId), ['1001'])
  assert.equal(charts.data.trend.reduce((sum: number, item: { count: number }) => sum + item.count, 0), 1)
  assert.deepEqual(charts.data.distribution.map((item: { command: string }) => item.command), ['a'])
  assert.deepEqual(charts.data.successRate, { success: 1, fail: 0 })
  assert.deepEqual(charts.data.userRank.map((item: { userId: string }) => item.userId), ['u1'])
})

test('legacy settings APIs require global console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/settings/get', {})
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/settings/update', { settings: {} })
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/settings/reset', {})
})

test('legacy log search filters records to the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const logs = await callListener(listeners, 'stuhelperGroupCenter/logs/search', {})
  assert.equal(logs.data.total, 1)
  assert.deepEqual(logs.data.list.map((item: { guildId: string }) => item.guildId), ['1001'])

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/logs/search', { guildId: '2002' })
})

test('legacy auth read APIs enforce the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerWebSocketAPI(ctx as any, service as any)

  const roles = await callListener(listeners, 'stuhelperGroupCenter/auth/user/get', { userId: 'target' })
  assert.deepEqual(roles.data, ['scoped-role'])

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/auth/role/members', { roleId: 'other-role' })
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/auth/users-by-authority', { authority: 4 })
})

test('role member import reports assignment failures instead of returning partial success', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  service.auth.assignRole = async (userId: string) => {
    if (userId === 'bad-user') {
      throw new Error('assignment failed for bad-user')
    }
  }

  registerWebSocketAPI(ctx as any, service as any)

  const result = await callListener(listeners, 'stuhelperGroupCenter/auth/role/import-members', {
    roleId: 'scoped-role',
    userIds: ['ok-user', 'bad-user'],
  })

  assert.equal(result.success, false)
  assert.match(result.error || '', /assignment failed for bad-user/)
})

async function assertRejectsScope(
  listeners: Map<string, Listener>,
  event: string,
  params: Record<string, string>,
) {
  const listener = listeners.get(event)
  assert.ok(listener, `${event} listener should be registered`)

  const result = await listener.call(createConsoleClient(), params)

  assert.equal(result.success, false)
  assert.match(result.error || '', /outside of the current console guild scope/)
}

async function assertRejectsGlobalScope(
  listeners: Map<string, Listener>,
  event: string,
  params: Record<string, unknown>,
) {
  const listener = listeners.get(event)
  assert.ok(listener, `${event} listener should be registered`)

  const result = await listener.call(createConsoleClient(), params)

  assert.equal(result.success, false)
  assert.match(result.error || '', /(requires global console scope|outside of the current console guild scope)/)
}

async function assertRejectsNotFound(
  listeners: Map<string, Listener>,
  event: string,
  params: Record<string, unknown>,
) {
  const listener = listeners.get(event)
  assert.ok(listener, `${event} listener should be registered`)

  const result = await listener.call(createConsoleClient(), params)

  assert.equal(result.success, false)
  assert.match(result.error || '', /订阅不存在/)
}

async function callListener(
  listeners: Map<string, Listener>,
  event: string,
  params: Record<string, unknown>,
) {
  const listener = listeners.get(event)
  assert.ok(listener, `${event} listener should be registered`)
  return listener.call(createConsoleClient(), params)
}

function createContext(listeners: Map<string, Listener>) {
  return {
    console: {
      addListener(event: string, callback: Listener) {
        listeners.set(event, callback)
      },
    },
    bots: [
      {
        platform: 'onebot',
        selfId: 'bot-1',
        getGuildMemberList: async () => ({ data: [] }),
        getGuild: async (guildId: string) => ({ id: guildId, name: `guild-${guildId}` }),
        sendMessage: async () => ['msg-1'],
        deleteMessage: async () => undefined,
      },
    ],
    database: {
      get: async (table: string) => {
        if (table === 'binding') {
          return [{ aid: 42, platform: 'onebot', pid: 'operator' }]
        }
        if (table === 'user') {
          return [{ id: 7, name: 'authority-user' }]
        }
        return []
      },
    },
    on() {},
    logger: () => ({
      debug() {},
      error() {},
      info() {},
      warn() {},
    }),
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

function createService(guildIds: string[]) {
  return {
    data: {
      groupConfig: {
        getAll: () => ({
          '1001': { enabled: true },
          '2002': { enabled: true },
        }),
        get: (guildId: string) => ({ guildId, enabled: true }),
        set() {},
        delete() {},
        reload() {},
        flush: async () => undefined,
      },
      warns: createMapStore({
        '1001': { u1: { count: 1, timestamp: 1 } },
        '2002': { u2: { count: 1, timestamp: 1 } },
      }),
      blacklist: createMapStore({
        u1: { userId: 'u1', guildId: '1001', timestamp: 1 },
        u2: { userId: 'u2', guildId: '2002', timestamp: 1 },
      }),
      authUsers: {
        flush: async () => undefined,
      },
      subscriptions: createMapStore({
        list: [
          { type: 'group', id: '1001', features: {} },
          { type: 'group', id: '2002', features: {} },
        ],
      }),
    },
    auth: {
      getRoles: () => [
        { id: 'scoped-role', guildIds },
        { id: 'other-role', guildIds: ['2002'] },
        { id: 'global-role', guildIds: [] },
      ],
      isBuiltinRole: () => false,
      getUserRoleIds: (userId: string) => {
        if (userId === 'onebot:operator' || userId === 'operator') {
          return ['scoped-role']
        }
        if (userId === 'target') {
          return ['scoped-role', 'other-role', 'global-role']
        }
        return []
      },
      getRoleMembers: (roleId: string) => {
        const members: Record<string, string[]> = {
          'scoped-role': ['u1'],
          'other-role': ['u2'],
          'global-role': ['u3'],
        }
        return members[roleId] || []
      },
      assignRole: async () => undefined,
    },
    cache: {
      getCachedData: () => ({ guilds: {}, members: {}, users: {} }),
      getGuildInfo: async (guildId: string) => ({ name: `guild-${guildId}` }),
      getMemberInfo: async (guildId: string, userId: string) => ({ name: `${guildId}-${userId}` }),
      getUserInfo: async (userId: string) => ({ name: userId }),
      getStats: () => ({ guilds: 2 }),
      refreshAll: async () => undefined,
      clearAll: async () => undefined,
    },
    getAllModules: () => [
      {
        meta: { name: 'log', description: 'log' },
        state: 'active',
        getAllLogs: async () => [
          { timestamp: new Date().toISOString(), command: 'a', success: true, guildId: '1001', userId: 'u1' },
          { timestamp: new Date().toISOString(), command: 'b', success: true, guildId: '2002', userId: 'u2' },
        ],
      },
    ],
    settings: {
      settings: {
        openai: {
          apiKey: 'secret',
        },
      },
      update: async () => undefined,
      reset: async () => undefined,
    },
  }
}

function createMapStore<T>(initial: Record<string, T>) {
  const values = { ...initial }
  return {
    getAll: () => values,
    get: (key: string) => values[key],
    set: (key: string, value: T) => {
      values[key] = value
    },
    delete: (key: string) => {
      delete values[key]
    },
    reload() {},
    flush: async () => undefined,
  }
}
