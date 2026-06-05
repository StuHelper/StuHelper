import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const servicesDir = dirname(fileURLToPath(import.meta.url))

test('cache service keeps Koishi logger typed explicitly', () => {
  const source = readFileSync(resolve(servicesDir, './cache.service.ts'), 'utf8')

  assert.doesNotMatch(source, /private logger: any/)
  assert.doesNotMatch(source, /^import \{ Context \} from 'koishi'/m)
  assert.match(source, /import type \{ Context, Logger \} from 'koishi'/)
  assert.match(source, /private readonly logger: Logger/)
})
