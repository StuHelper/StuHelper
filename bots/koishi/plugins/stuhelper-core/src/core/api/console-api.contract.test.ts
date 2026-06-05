import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const apiDir = dirname(fileURLToPath(import.meta.url))

function readApiFile(relativePath: string): string {
  return readFileSync(resolve(apiDir, relativePath), 'utf8')
}

test('log and stats console APIs use typed log module data', () => {
  const logsSource = readApiFile('./logs-api.ts')
  const statsSource = readApiFile('./stats-api.ts')
  const scopeFiltersSource = readApiFile('./scope-filters.ts')

  assert.doesNotMatch(logsSource, /as any/)
  assert.doesNotMatch(statsSource, /as any/)
  assert.doesNotMatch(logsSource, /log: any/)
  assert.doesNotMatch(statsSource, /logs: any\[\]/)
  assert.doesNotMatch(logsSource, /getAllModules\(\)\.find\(\(module\) => module\.meta\.name === 'log'\) as any/)
  assert.doesNotMatch(statsSource, /getAllModules\(\)\.find\(\(module\) => module\.meta\.name === 'log'\) as any/)
  assert.match(logsSource, /readCommandLogs\(api\)/)
  assert.match(statsSource, /readCommandLogs\(api\)/)
  assert.match(scopeFiltersSource, /filterLogs<T extends \{ guildId\?: unknown \}>/)
})

test('auth console API reuses shared guild admin role detection', () => {
  const authSource = readApiFile('./auth-api.ts')

  assert.match(authSource, /isGuildAdminMember/)
  assert.match(authSource, /filter\(isGuildAdminMember\)/)
  assert.doesNotMatch(authSource, /function isGuildAdmin\(/)
  assert.doesNotMatch(authSource, /roles\.includes\('admin'\)/)
  assert.doesNotMatch(authSource, /roles\.includes\('owner'\)/)
})
