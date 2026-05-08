import assert from 'node:assert/strict'
import test from 'node:test'

import { registerMemberManageCommands, parseKickInput } from './member-manage-commands.ts'

test('parseKickInput accepts command options and explicit guild id for blacklist kick', () => {
  assert.deepEqual(parseKickInput('10001', 'default-guild', { black: true }), {
    userId: '10001',
    targetGroup: 'default-guild',
    black: true,
  })

  assert.deepEqual(parseKickInput('10001 -b target-guild', 'default-guild'), {
    userId: '10001',
    targetGroup: 'target-guild',
    black: true,
  })
})

test('kick -b reports partial failure when backend blacklist creation fails', async () => {
  const originalFetch = globalThis.fetch
  const calls: Array<{ readonly path: string; readonly init?: RequestInit }> = []
  globalThis.fetch = async (input, init) => {
    calls.push({ path: String(input), init })
    return new Response(JSON.stringify({
      success: false,
      error: { code: 'BLACKLIST_FAILED', message: 'backend down' },
    }), {
      status: 500,
      headers: { 'content-type': 'application/json' },
    })
  }

  try {
    const commandActions = new Map<string, Function>()
    const logs: unknown[] = []
    const host = {
      config: {},
      platformConfig: { baseUrl: 'https://platform.example', serviceToken: 'token' },
      ctx: {
        stuhelperGroupCenter: {
          pushMessage: async () => undefined,
        },
      },
      logCommand: (entry: unknown) => logs.push(entry),
      registerCommand(def: { readonly name: string }) {
        const chain = {
          example: () => chain,
          option: () => chain,
          action(fn: Function) {
            commandActions.set(def.name, fn)
            return chain
          },
        }
        return chain
      },
    }
    registerMemberManageCommands(host as any)

    const kicked: unknown[] = []
    const session = {
      platform: 'qq',
      guildId: 'source-guild',
      userId: 'operator-qq',
      content: 'kick 10001 -b target-guild',
      bot: {
        kickGuildMember: async (...args: unknown[]) => kicked.push(args),
      },
    }

    const result = await commandActions.get('kick')?.({ session, options: {} }, '10001 -b target-guild')

    assert.equal(result, '已把 10001 踢出群 target-guild，但加入黑名单失败：backend down')
    assert.deepEqual(kicked, [['target-guild', '10001', true]])
    assert.equal(calls.length, 1)
    assert.match(calls[0].path, /\/api\/v1\/bot\/member-blacklist$/)
    assert.deepEqual(logs.at(-1), {
      session,
      command: 'kick',
      target: '10001',
      result: '部分成功：已踢出但加入黑名单失败：backend down',
      success: false,
    })
  } finally {
    globalThis.fetch = originalFetch
  }
})
