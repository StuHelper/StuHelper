import assert from 'node:assert/strict'
import test from 'node:test'

import { registerWebSocketAPI } from './index'

type Listener = (params: any) => Promise<{ success: boolean; error?: string }>
type EventHandler = (session: any) => Promise<void>

const CHAT_IMAGE_URL = 'https://gchat.qpic.cn/gchatpic_new/1/2-3-ABC/0'
const CHAT_IMAGE_FILE = '6B4DE3DFD1BD271E3297859D41C530F5.jpg'
const TEST_PLATFORM_CONFIG = {
  baseUrl: 'http://platform.test',
  serviceToken: 'test-token',
}

test('chat guild APIs reject guilds outside the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/chat/guild-members', { guildId: '2002' })
  await assertRejectsScope(listeners, 'stuhelperGroupCenter/chat/guild-info', { guildId: '2002' })
})

test('chat guild member list recognizes universal owner and admin role objects', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  ctx.bots[0].getGuildMemberList = async () => ({
    data: [
      { user: { id: 'member-1', name: 'Normal' }, roles: [{ id: 'member' }] },
      { user: { id: 'owner-1', name: 'Owner' }, roles: [{ id: 'owner' }] },
      { user: { id: 'admin-1', name: 'Admin' }, roles: [{ id: 'admin' }] },
    ],
  })
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

  const result = await callListener(listeners, 'stuhelperGroupCenter/chat/guild-members', { guildId: '1001' })

  assert.equal(result.success, true)
  assert.deepEqual(result.data.members.map((member: { id: string }) => member.id), ['owner-1', 'admin-1', 'member-1'])
  assert.deepEqual(
    result.data.members.map((member: { isOwner: boolean; isAdmin: boolean }) => ({
      isOwner: member.isOwner,
      isAdmin: member.isAdmin,
    })),
    [
      { isOwner: true, isAdmin: true },
      { isOwner: false, isAdmin: true },
      { isOwner: false, isAdmin: false },
    ],
  )
})

test('config and role read APIs filter data to the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

  const configs = await callListener(listeners, 'stuhelperGroupCenter/config/list', {})
  assert.deepEqual(Object.keys(configs.data || {}), ['1001'])

  const config = await callListener(listeners, 'stuhelperGroupCenter/config/get', { guildId: '2002' })
  assert.equal(config.success, false)
  assert.match(config.error || '', /outside of the current console guild scope/)

  const roles = await callListener(listeners, 'stuhelperGroupCenter/auth/role/list', {})
  assert.deepEqual((roles.data || []).map((role: { id: string }) => role.id), ['scoped-role'])
})

test('config update validates payloads before writing group config', async () => {
  const listeners = new Map<string, Listener>()
  const writes: Array<{ guildId: string; config: Record<string, unknown> }> = []
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  service.data.groupConfig.set = (guildId: string, config: Record<string, unknown>) => {
    writes.push({ guildId, config })
  }

  registerTestWebSocketAPI(ctx as any, service as any)

  const invalidArray = await callListener(listeners, 'stuhelperGroupCenter/config/update', {
    guildId: '1001',
    config: [],
  })
  assert.equal(invalidArray.success, false)
  assert.match(invalidArray.error || '', /group config must be an object/)

  const unsafeKey = await callListener(listeners, 'stuhelperGroupCenter/config/update', {
    guildId: '1001',
    config: { constructor: { prototype: { polluted: true } } },
  })
  assert.equal(unsafeKey.success, false)
  assert.match(unsafeKey.error || '', /unsupported key: constructor/)

  const valid = await callListener(listeners, 'stuhelperGroupCenter/config/update', {
    guildId: ' 1001 ',
    config: {
      welcomeEnabled: true,
      keywords: ['入群'],
      ignoredUndefined: undefined,
      forbidden: {
        autoDelete: false,
        autoBan: false,
        autoKick: false,
        muteDuration: 600000,
      },
    },
  })

  assert.equal(valid.success, true)
  assert.deepEqual(writes, [{
    guildId: '1001',
    config: {
      welcomeEnabled: true,
      keywords: ['入群'],
      forbidden: {
        autoDelete: false,
        autoBan: false,
        autoKick: false,
        muteDuration: 600000,
      },
    },
  }])
})

test('chat write APIs reject messages outside the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

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

test('chat image fetch requires a delivered OneBot image file', async () => {
  const listeners = new Map<string, Listener>()
  const imageFetches: string[] = []
  const ctx = createContext(listeners)
  ctx.bots[0].internal = {
    getImage: async (file: string) => {
      imageFetches.push(file)
      return { base64: 'AAAA', file_name: file }
    },
  }
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

  const result = await callListener(listeners, 'stuhelperGroupCenter/image/fetch', {
    url: CHAT_IMAGE_URL,
    file: CHAT_IMAGE_FILE,
  })

  assert.equal(result.success, false)
  assert.match(result.error || '', /not attached to a delivered chat message/)
  assert.deepEqual(imageFetches, [])
})

test('chat image fetch passes only the OneBot file identifier to get_image', async () => {
  const listeners = new Map<string, Listener>()
  const events = new Map<string, EventHandler[]>()
  const imageFetches: string[] = []
  const ctx = createContext(listeners, events)
  ctx.bots[0].internal = {
    getImage: async (file: string) => {
      imageFetches.push(file)
      return { base64: 'AAAA', file_name: file }
    },
  }
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)
  await emit(events, 'message', createImageMessageSession('1001'))

  const result = await callListener(listeners, 'stuhelperGroupCenter/image/fetch', {
    url: CHAT_IMAGE_URL,
    file: CHAT_IMAGE_FILE,
  })

  assert.equal(result.success, true)
  assert.equal(result.data?.mimeType, 'image/jpeg')
  assert.deepEqual(imageFetches, [CHAT_IMAGE_FILE])
})

test('chat image fetch enforces the guild scope recorded from delivered messages', async () => {
  const listeners = new Map<string, Listener>()
  const events = new Map<string, EventHandler[]>()
  const imageFetches: string[] = []
  const ctx = createContext(listeners, events)
  ctx.bots[0].internal = {
    getImage: async (file: string) => {
      imageFetches.push(file)
      return { base64: 'AAAA', file_name: file }
    },
  }
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)
  await emit(events, 'message', createImageMessageSession('2002'))

  const result = await callListener(listeners, 'stuhelperGroupCenter/image/fetch', {
    url: CHAT_IMAGE_URL,
    file: CHAT_IMAGE_FILE,
  })

  assert.equal(result.success, false)
  assert.match(result.error || '', /outside of the current console guild scope/)
  assert.deepEqual(imageFetches, [])
})

test('chat image fetch does not derive OneBot file identifiers from URLs', async () => {
  const listeners = new Map<string, Listener>()
  const events = new Map<string, EventHandler[]>()
  const imageFetches: string[] = []
  const ctx = createContext(listeners, events)
  ctx.bots[0].internal = {
    getImage: async (file: string) => {
      imageFetches.push(file)
      return { base64: 'AAAA', file_name: file }
    },
  }
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)
  await emit(events, 'message', createImageMessageSession('1001'))

  const result = await callListener(listeners, 'stuhelperGroupCenter/image/fetch', {
    url: CHAT_IMAGE_URL,
  })

  assert.equal(result.success, false)
  assert.match(result.error || '', /image file is required/)
  assert.deepEqual(imageFetches, [])
})

test('chat send rejects oversized content', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

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

  registerTestWebSocketAPI(ctx as any, service as any)

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

  registerTestWebSocketAPI(ctx as any, service as any)

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

  registerTestWebSocketAPI(ctx as any, service as any)

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

  registerTestWebSocketAPI(ctx as any, service as any)

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
  const restoreFetch = mockPlatformFetch([])

  try {
    registerTestWebSocketAPI(ctx as any, service as any)

    const dashboard = await callListener(listeners, 'stuhelperGroupCenter/stats/dashboard', {})
    assert.equal(dashboard.data.totalGroups, 1)
    assert.equal(dashboard.data.totalWarns, 1)
    assert.equal(dashboard.data.totalBlacklisted, 2)
    assert.equal(dashboard.data.totalSubscriptions, 1)

    const charts = await callListener(listeners, 'stuhelperGroupCenter/stats/charts', {})
    assert.deepEqual(charts.data.guildRank.map((item: { guildId: string }) => item.guildId), ['1001'])
    assert.equal(charts.data.trend.reduce((sum: number, item: { count: number }) => sum + item.count, 0), 1)
    assert.deepEqual(charts.data.distribution.map((item: { command: string }) => item.command), ['a'])
    assert.deepEqual(charts.data.successRate, { success: 1, fail: 0 })
    assert.deepEqual(charts.data.userRank.map((item: { userId: string }) => item.userId), ['u1'])
  } finally {
    restoreFetch()
  }
})

test('blacklist console APIs use backend member blacklist and enforce release scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  const fetchCalls: Array<{ url: string; init?: RequestInit }> = []
  const restoreFetch = mockPlatformFetch(fetchCalls)

  try {
    registerWebSocketAPI(ctx as any, service as any, TEST_PLATFORM_CONFIG)

    const list = await callListener(listeners, 'stuhelperGroupCenter/blacklist/list', {})
    assert.equal(list.success, true)
    assert.deepEqual(
      list.data.list.map((item: { id: string }) => item.id),
      ['blk-global', 'blk-1001'],
    )

    const blocked = await callListener(listeners, 'stuhelperGroupCenter/blacklist/remove', {
      id: 'blk-2002',
      releaseReasonCode: 'manual_pardon',
      releaseReason: 'wrong scope',
    })
    assert.equal(blocked.success, false)
    assert.match(blocked.error || '', /outside of the current console scope/)

    const released = await callListener(listeners, 'stuhelperGroupCenter/blacklist/remove', {
      id: 'blk-1001',
      releaseReasonCode: 'manual_pardon',
      releaseReason: 'manual release',
    })
    assert.equal(released.success, true)
    assert.equal(fetchCalls.at(-1)?.url, 'http://platform.test/api/v1/bot/member-blacklist/blk-1001/release')
  } finally {
    restoreFetch()
  }
})

test('blacklist console add rejects guild scoped entries without guildID', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  const fetchCalls: Array<{ url: string; init?: RequestInit }> = []
  const restoreFetch = mockPlatformFetch(fetchCalls)

  try {
    registerWebSocketAPI(ctx as any, service as any, TEST_PLATFORM_CONFIG)

    const result = await callListener(listeners, 'stuhelperGroupCenter/blacklist/add', {
      platform: 'qq',
      subjectID: '10001',
      scopeType: 'guild',
      reasonText: 'missing guild',
    })

    assert.equal(result.success, false)
    assert.match(result.error || '', /guildID is required/)
    assert.equal(fetchCalls.length, 0)
  } finally {
    restoreFetch()
  }
})

test('legacy settings APIs require global console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/settings/get', {})
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/settings/update', { settings: {} })
  await assertRejectsGlobalScope(listeners, 'stuhelperGroupCenter/settings/reset', {})
})

test('legacy log search filters records to the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

  const logs = await callListener(listeners, 'stuhelperGroupCenter/logs/search', {})
  assert.equal(logs.data.total, 1)
  assert.deepEqual(logs.data.list.map((item: { guildId: string }) => item.guildId), ['1001'])

  await assertRejectsScope(listeners, 'stuhelperGroupCenter/logs/search', { guildId: '2002' })
})

test('legacy log search falls back to data store when log runtime module is absent', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  service.getAllModules = () => []
  service.data.commandLogs = createMapStore({
    logs: [
      {
        timestamp: 1_787_803_200_000,
        guildId: '1001',
        userId: 'u1',
        command: 'fallback',
        target: 'command',
        details: 'loaded from data store',
      },
      {
        timestamp: 1_787_803_201_000,
        guildId: '2002',
        userId: 'u2',
        command: 'hidden',
        target: 'command',
        details: 'outside scope',
      },
    ],
  })

  registerTestWebSocketAPI(ctx as any, service as any)

  const logs = await callListener(listeners, 'stuhelperGroupCenter/logs/search', {})

  assert.equal(logs.success, true)
  assert.equal(logs.data.total, 1)
  assert.equal(logs.data.list[0].command, 'fallback')
  assert.equal(logs.data.list[0].result, 'loaded from data store')
})

test('legacy log search falls back when named log module lookup throws', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  Object.assign(service, {
    getModule(name: string) {
      throw new Error(`${name[0].toUpperCase()}${name.slice(1)} module not found`)
    },
  })
  service.getAllModules = () => []
  service.data.commandLogs = createMapStore({
    logs: [
      {
        timestamp: 1_787_803_200_000,
        guildId: '1001',
        userId: 'u1',
        command: 'fallback',
        target: 'command',
        details: 'loaded despite missing module lookup',
      },
    ],
  })

  registerTestWebSocketAPI(ctx as any, service as any)

  const logs = await callListener(listeners, 'stuhelperGroupCenter/logs/search', {})

  assert.equal(logs.success, true)
  assert.equal(logs.data.total, 1)
  assert.equal(logs.data.list[0].command, 'fallback')
  assert.equal(logs.data.list[0].result, 'loaded despite missing module lookup')
})

test('chart stats falls back to data store when log runtime module is absent', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])
  service.getAllModules = () => []
  service.data.commandLogs = createMapStore({
    logs: [
      {
        timestamp: Date.now(),
        guildId: '1001',
        userId: 'u1',
        command: 'fallback',
        target: 'command',
        details: 'loaded from data store',
      },
    ],
  })

  registerTestWebSocketAPI(ctx as any, service as any)

  const charts = await callListener(listeners, 'stuhelperGroupCenter/stats/charts', { days: 1 })

  assert.equal(charts.success, true)
  assert.deepEqual(charts.data.distribution, [{ command: 'fallback', count: 1 }])
  assert.deepEqual(charts.data.successRate, { success: 1, fail: 0 })
})

test('legacy auth read APIs enforce the console scope', async () => {
  const listeners = new Map<string, Listener>()
  const ctx = createContext(listeners)
  const service = createService(['1001'])

  registerTestWebSocketAPI(ctx as any, service as any)

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

  registerTestWebSocketAPI(ctx as any, service as any)

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

function registerTestWebSocketAPI(ctx: unknown, service: unknown) {
  registerWebSocketAPI(ctx as any, service as any, TEST_PLATFORM_CONFIG)
}

function createContext(
  listeners: Map<string, Listener>,
  events: Map<string, EventHandler[]> = new Map(),
) {
  return {
    console: {
      clients: {},
      addListener(event: string, callback: Listener) {
        listeners.set(event, callback)
      },
    },
    bots: [
      {
        platform: 'onebot',
        selfId: 'bot-1',
        internal: undefined as undefined | { getImage(file: string): Promise<unknown> },
        getGuildMemberList: async () => ({ data: [] }),
        getGuild: async (guildId: string) => ({ id: guildId, name: `guild-${guildId}` }),
        getGuildMember: async (_guildId: string, userId: string) => ({ user: { name: userId } }),
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
    on(event: string, handler: EventHandler) {
      const handlers = events.get(event) ?? []
      handlers.push(handler)
      events.set(event, handlers)
    },
    logger: () => ({
      debug() {},
      error() {},
      info() {},
      warn() {},
    }),
  }
}

async function emit(events: Map<string, EventHandler[]>, event: string, session: any) {
  for (const handler of events.get(event) ?? []) {
    await handler(session)
  }
}

function createImageMessageSession(guildId: string) {
  return {
    platform: 'onebot',
    selfId: 'bot-1',
    guildId,
    channelId: guildId,
    messageId: `msg-${guildId}`,
    timestamp: 1,
    userId: 'u1',
    author: { name: 'u1' },
    content: '',
    elements: [
      {
        type: 'img',
        attrs: {
          src: CHAT_IMAGE_URL,
          file: CHAT_IMAGE_FILE,
        },
      },
    ],
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

function mockPlatformFetch(calls: Array<{ url: string; init?: RequestInit }>) {
  const originalFetch = globalThis.fetch
  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input)
    calls.push({ url, init })
    if (url.includes('/api/v1/bot/member-blacklist?')) {
      const search = new URL(url).searchParams
      const scopeType = search.get('scopeType')
      const guildID = search.get('guildID')
      const entries = [
        memberBlacklistEntry({ id: 'blk-global', subjectID: 'global-user', scopeType: 'global' }),
        memberBlacklistEntry({ id: 'blk-1001', subjectID: 'guild-user', guildID: '1001' }),
        memberBlacklistEntry({ id: 'blk-2002', subjectID: 'other-user', guildID: '2002' }),
      ].filter((entry) => {
        if (scopeType === 'global') return entry.scopeType === 'global'
        if (scopeType === 'guild') return entry.scopeType === 'guild' && entry.guildID === guildID
        return true
      })
      return jsonResponse({
        data: { list: entries, total: entries.length },
      })
    }
    if (url.includes('/api/v1/bot/member-blacklist/')) {
      return jsonResponse({
        data: memberBlacklistEntry({ id: 'blk-1001', subjectID: 'guild-user', guildID: '1001' }),
      })
    }
    return jsonResponse({ success: true })
  }
  return () => {
    globalThis.fetch = originalFetch
  }
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })
}

function memberBlacklistEntry(input: {
  id: string
  subjectID: string
  scopeType?: 'global' | 'guild'
  guildID?: string
}) {
  return {
    id: input.id,
    platform: 'onebot',
    subjectType: 'qq_user',
    subjectID: input.subjectID,
    scopeType: input.scopeType || 'guild',
    guildID: input.guildID,
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: 'test',
    createdByType: 'service_account',
    createdByID: 'koishi-runtime',
    createdFrom: 'koishi_console',
    metadata: {},
    createdAt: '2026-05-08T00:00:00Z',
    updatedAt: '2026-05-08T00:00:00Z',
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
      commandLogs: createMapStore({
        logs: [],
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
