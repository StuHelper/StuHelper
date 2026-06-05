import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { registerStatusCommands, type StatusCommandHost } from './status-commands'

type CommandAction = () => Promise<unknown>

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('status command normalizes unknown caught errors', () => {
  const source = readFileSync(resolve(modulesDir, './status-commands.ts'), 'utf8')

  assert.doesNotMatch(source, /\(error as Error\)\.message/)
  assert.match(source, /commandErrorMessage/)
})

test('gstatus reports non-Error status data failures without undefined output', async () => {
  const commandActions = new Map<string, CommandAction>()
  const host = {
    ctx: {
      puppeteer: {
        page: async () => {
          throw new Error('puppeteer should not be reached')
        },
      },
      registry: {
        size: 0,
      },
    },
    data: {
      groupConfig: {
        getAll: async () => {
          throw 'store offline'
        },
      },
      commandLogs: {
        getAll: async () => [],
      },
    },
    registerCommand(def: { readonly name: string }) {
      const chain = {
        action(fn: CommandAction) {
          commandActions.set(def.name, fn)
          return chain
        },
      }
      return chain
    },
  }
  registerStatusCommands(host as unknown as StatusCommandHost)

  assert.equal(await commandActions.get('gstatus')?.(), '生成状态图失败：store offline')
})
