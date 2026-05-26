import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const modulesDir = dirname(fileURLToPath(import.meta.url))

function readModuleFile(relativePath: string): string {
  return readFileSync(resolve(modulesDir, relativePath), 'utf8')
}

test('log command helpers use explicit option types and unknown error boundaries', () => {
  const commandSource = readModuleFile('./log-command-commands.ts')
  const operationSource = readModuleFile('./log-operation-commands.ts')
  const formatterSource = readModuleFile('./log-formatters.ts')

  for (const source of [commandSource, operationSource, formatterSource]) {
    assert.doesNotMatch(source, /options: any/)
    assert.doesNotMatch(source, /catch \(error: any\)/)
    assert.doesNotMatch(source, /\$\{error\.message\}/)
  }

  assert.match(commandSource, /interface CommandLogCheckOptions extends LogFilterOptions/)
  assert.match(commandSource, /interface CommandLogStatsOptions extends LogStatsOptions/)
  assert.match(operationSource, /interface ClearOperationLogOptions/)
  assert.match(formatterSource, /export interface LogFilterOptions/)
  assert.match(formatterSource, /export interface LogStatsOptions extends LogFilterOptions/)
})
