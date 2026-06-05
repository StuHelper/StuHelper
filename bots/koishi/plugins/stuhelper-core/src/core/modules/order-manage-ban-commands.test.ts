import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { registerOrderBanCommands } from './order-manage-ban-commands'

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('order and member manage commands normalize unknown caught errors', () => {
  const sources = [
    readFileSync(resolve(modulesDir, './order-manage-ban-commands.ts'), 'utf8'),
    readFileSync(resolve(modulesDir, './order-manage-group-commands.ts'), 'utf8'),
    readFileSync(resolve(modulesDir, './member-manage-commands.ts'), 'utf8'),
  ]

  for (const source of sources) {
    assert.doesNotMatch(source, /error\.message/)
    assert.match(source, /commandErrorMessage/)
  }
})

test('ban command reports non-Error adapter failures without undefined output', async () => {
  const commandActions = new Map<string, Function>()
  const logs: unknown[] = []
  const host = {
    logCommand: (entry: unknown) => logs.push(entry),
    recordMute: () => undefined,
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
  }
  registerOrderBanCommands(host as any)

  const session = {
    guildId: 'source-guild',
    bot: {
      muteGuildMember: async () => {
        throw 'adapter offline'
      },
    },
  }

  const result = await commandActions.get('ban')?.({ session }, '10001 1m target-guild')

  assert.equal(result, '喵呜...禁言失败了：adapter offline')
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'ban',
    target: '10001',
    result: '失败：未知错误',
    success: false,
  })
})
