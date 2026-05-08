import assert from 'node:assert/strict'
import test from 'node:test'

import { registerMemberManageCommands, parseKickInput } from './member-manage-commands.ts'

test('parseKickInput accepts command options and explicit guild id for blacklist kick', () => {
  assert.deepEqual(parseKickInput('10001', 'default-guild', { black: true }), {
    userId: '10001',
    targetGroup: 'default-guild',
    black: true,
    global: false,
  })

  assert.deepEqual(parseKickInput('10001 -b --global target-guild', 'default-guild'), {
    userId: '10001',
    targetGroup: 'target-guild',
    black: true,
    global: true,
  })
})

test('kick -b reports partial failure when backend blacklist creation fails', async () => {
  const commandActions = new Map<string, Function>()
  const logs: unknown[] = []
  const blacklistRequests: unknown[] = []
  const host = {
    config: {},
    memberBlacklistBackend: {
      async createMemberBlacklist(input: unknown) {
        blacklistRequests.push(input)
        throw new Error('backend down')
      },
    },
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
  assert.equal(blacklistRequests.length, 1)
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'kick',
    target: '10001',
    result: '部分成功：已踢出但加入黑名单失败：backend down',
    success: false,
  })
})
