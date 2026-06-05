import assert from 'node:assert/strict'
import test from 'node:test'

import type { Role } from '../../types'
import type { WebSocketAPIContext } from './api-context'
import type { ChatImageAccessRegistry } from './chat-image-fetch'
import { registerChatMessageBroadcast } from './chat-message-broadcast'

type BroadcastHandler = (session: Record<string, unknown>) => Promise<void> | void

interface SentPayload {
  readonly type: string
  readonly body: Record<string, unknown>
}

test('chat message broadcast keeps element-only payload content as a string', async () => {
  const harness = createBroadcastHarness()

  await harness.emit('message', {
    platform: 'onebot',
    selfId: 'bot-1',
    guildId: '1001',
    channelId: '1001',
    userId: 'u1',
    id: 42,
    timestamp: 1,
    elements: [{ type: 'text', attrs: { content: 'element text' } }],
  })

  assert.equal(harness.sent.length, 1)
  assert.equal(harness.sent[0].body.id, '42')
  assert.equal(harness.sent[0].body.content, '')
  assert.equal(typeof harness.sent[0].body.content, 'string')
  assert.deepEqual(harness.remembered[0].elements, [{ type: 'text', attrs: { content: 'element text' } }])
})

test('chat message broadcast falls back to quote element preview when quote content is not text', async () => {
  const harness = createBroadcastHarness()

  await harness.emit('message', {
    platform: 'onebot',
    selfId: 'bot-1',
    guildId: '1001',
    channelId: '1001',
    userId: 'u1',
    messageId: 'msg-with-quote',
    timestamp: 2,
    content: 'reply',
    quote: {
      id: 'quoted-msg',
      content: { unsafe: true },
      elements: [{ type: 'text', attrs: { content: 'quoted fallback' } }],
      user: { id: 'u2' },
    },
  })

  const elements = harness.sent[0].body.elements as Array<{ type?: string; attrs?: Record<string, unknown> }>
  assert.equal(elements[0].type, 'quote')
  assert.equal(elements[0].attrs?.id, 'quoted-msg')
  assert.equal(elements[0].attrs?.user, 'u2')
  assert.equal(elements[0].attrs?.content, 'quoted fallback')
})

test('chat message broadcast drops malformed elements before serializing payload content', async () => {
  const harness = createBroadcastHarness()

  await harness.emit('message', {
    platform: 'onebot',
    selfId: 'bot-1',
    guildId: '1001',
    channelId: '1001',
    userId: 'u1',
    messageId: 'msg-malformed-element',
    timestamp: 3,
    elements: [
      { type: 'img', attrs: 'not-object' },
      { type: 'img', attrs: { src: 'https://example.com/a.png', file: 'a.png' } },
    ],
  })

  const elements = harness.sent[0].body.elements as Array<{ type?: string; attrs?: Record<string, unknown> }>
  assert.deepEqual(elements, [{ type: 'img', attrs: { src: 'https://example.com/a.png', file: 'a.png' } }])
  assert.match(String(harness.sent[0].body.content), /https:\/\/example\.com\/a\.png/)
  assert.deepEqual(harness.remembered[0].elements, elements)
})

function createBroadcastHarness() {
  const handlers = new Map<string, BroadcastHandler[]>()
  const sent: SentPayload[] = []
  const remembered: Array<{ elements: unknown; guildId: string | undefined }> = []

  const api = {
    ctx: {
      console: {
        clients: {
          admin: {
            id: 'admin',
            auth: { id: 1, authority: 4 },
            send(payload: SentPayload) {
              sent.push(payload)
            },
          },
        },
      },
      bots: [],
      database: {
        get: async () => [],
      },
      on(event: string, handler: BroadcastHandler) {
        const list = handlers.get(event) ?? []
        list.push(handler)
        handlers.set(event, list)
      },
      logger: () => ({
        debug() {},
        error() {},
        info() {},
        warn() {},
      }),
    },
    service: {
      auth: {
        getRoles: (): Pick<Role, 'id' | 'guildIds'>[] => [],
        getUserRoleIds: () => [],
      },
    },
  } as unknown as WebSocketAPIContext

  const imageAccess = {
    remember(elements: unknown, guildId: string | undefined) {
      remembered.push({ elements, guildId })
    },
    assertAllowed() {
      throw new Error('not used')
    },
  } as unknown as ChatImageAccessRegistry

  registerChatMessageBroadcast(api, imageAccess)

  return {
    sent,
    remembered,
    async emit(event: string, session: Record<string, unknown>) {
      for (const handler of handlers.get(event) ?? []) {
        await handler(session)
      }
    },
  }
}
