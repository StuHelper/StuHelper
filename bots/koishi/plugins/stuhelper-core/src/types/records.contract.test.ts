import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const typesDir = dirname(fileURLToPath(import.meta.url))

test('record message elements use Koishi element types instead of any arrays', () => {
  const source = readFileSync(resolve(typesDir, './records.ts'), 'utf8')

  assert.doesNotMatch(source, /elements\?: any\[]/)
  assert.match(source, /import type \{ Element \} from 'koishi'/)
  assert.match(source, /elements\?: Element\[]/)
})
