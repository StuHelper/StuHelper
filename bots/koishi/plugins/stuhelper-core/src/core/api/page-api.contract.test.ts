import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const apiDir = dirname(fileURLToPath(import.meta.url))

function readApiFile(relativePath: string): string {
  return readFileSync(resolve(apiDir, relativePath), 'utf8')
}

test('entity-profile page listener applies console guild scope before loading profile data', () => {
  const source = readApiFile('./page-api.ts')
  const listenerStart = source.indexOf("stuhelperGroupCenter/page/entity-profile")
  const scopeIndex = source.indexOf('resolveRequiredConsoleGuildScope', listenerStart)
  const profileIndex = source.indexOf('entityPage.getProfile', listenerStart)

  assert.notEqual(listenerStart, -1)
  assert.ok(scopeIndex > listenerStart)
  assert.ok(profileIndex > scopeIndex)
  assert.match(source, /entityPage\.getProfile\(\{[\s\S]*\}, toEntityProfileScope\(scope\)\)/)
  assert.match(source, /return \{ guildIds: null \}/)
  assert.match(source, /return \{ guildIds: scope\.guildIds \}/)
})
