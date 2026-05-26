import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Context } from 'koishi'

import type { DataManager } from '../data'
import { getSystemStatusData } from './status-data'

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('help and status modules keep command help and status data typed', () => {
  const helpSource = readModuleFile('./help.module.ts')
  const statusSource = readModuleFile('./status-data.ts')

  assert.doesNotMatch(helpSource, /session: any/)
  assert.doesNotMatch(helpSource, /commands: any\[\]/)
  assert.doesNotMatch(helpSource, /cmd: any/)
  assert.doesNotMatch(helpSource, /\$\{e\.message\}/)
  assert.match(helpSource, /formatCommandHelpLine\(cmd: RegisteredCommand\)/)

  assert.doesNotMatch(statusSource, /as any/)
  assert.doesNotMatch(statusSource, /logCount: unknown/)
  assert.match(statusSource, /logCount: commandLogs\.logs\.length/)
})

test('getSystemStatusData counts command logs from the typed logs collection', async () => {
  const result = await getSystemStatusData(
    { registry: { size: 7 } } as unknown as Context,
    {
      groupConfig: {
        getAll: () => ({
          guildA: {},
          guildB: {},
        }),
      },
      commandLogs: {
        getAll: () => ({
          logs: [
            { timestamp: 1, guildId: 'guildA', userId: '1001', command: 'help', target: '', details: '' },
            { timestamp: 2, guildId: 'guildB', userId: '1002', command: 'gstatus', target: '', details: '' },
          ],
        }),
      },
    } as unknown as DataManager,
  )

  assert.equal(result.bot.plugins, 7)
  assert.equal(result.stuhelperGroupCenter.groupCount, 2)
  assert.equal(result.stuhelperGroupCenter.logCount, 2)
})

function readModuleFile(relativePath: string): string {
  return readFileSync(resolve(modulesDir, relativePath), 'utf8')
}
