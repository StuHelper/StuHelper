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
      get: async () => [{ platform: 'onebot', pid: 'operator' }],
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
        return []
      },
    },
    cache: {
      getCachedData: () => ({ guilds: {} }),
    },
  }
}
