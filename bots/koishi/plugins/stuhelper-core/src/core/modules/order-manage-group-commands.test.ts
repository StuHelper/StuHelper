import assert from 'node:assert/strict'
import test from 'node:test'

import { registerOrderGroupCommands } from './order-manage-group-commands'

test('nickname command accepts structured Koishi user arguments', async () => {
  const commandActions = new Map<string, Function>()
  const logs: unknown[] = []
  const groupCards: unknown[] = []
  const host = {
    logCommand: (entry: unknown) => logs.push(entry),
    registerCommand(def: { readonly name: string }) {
      const chain = {
        example: () => chain,
        action(fn: Function) {
          commandActions.set(def.name, fn)
          return chain
        },
      }
      return chain
    },
    data: {
      mutes: {
        getAll: () => ({}),
      },
    },
  }
  registerOrderGroupCommands(host as any)

  const session = {
    guildId: 'guild-1',
    bot: {
      platform: 'onebot',
      internal: {
        setGroupCard: async (...args: unknown[]) => groupCards.push(args),
      },
    },
  }

  const result = await commandActions.get('nickname')?.(
    { session },
    { id: 'onebot:10004' },
    '小猫咪',
  )

  assert.equal(result, '已将 10004 的昵称设置为 "小猫咪" 喵~')
  assert.deepEqual(groupCards, [['guild-1', '10004', '小猫咪']])
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'nickname',
    target: '10004',
    result: '成功：已设置昵称为 小猫咪, 群号 guild-1',
  })
})

test('nickname command reports non-Error adapter failures without undefined output', async () => {
  const commandActions = new Map<string, Function>()
  const logs: unknown[] = []
  const host = {
    logCommand: (entry: unknown) => logs.push(entry),
    registerCommand(def: { readonly name: string }) {
      const chain = {
        example: () => chain,
        action(fn: Function) {
          commandActions.set(def.name, fn)
          return chain
        },
      }
      return chain
    },
    data: {
      mutes: {
        getAll: () => ({}),
      },
    },
  }
  registerOrderGroupCommands(host as any)

  const session = {
    guildId: 'guild-1',
    bot: {
      platform: 'onebot',
      internal: {
        setGroupCard: async () => {
          throw 'adapter offline'
        },
      },
    },
  }

  const result = await commandActions.get('nickname')?.(
    { session },
    { id: 'onebot:10004' },
    '小猫咪',
  )

  assert.equal(result, '喵呜...设置昵称失败了：adapter offline')
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'nickname',
    target: '10004',
    result: '失败：未知错误',
    success: false,
  })
})
