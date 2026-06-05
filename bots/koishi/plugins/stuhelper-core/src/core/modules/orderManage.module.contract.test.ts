import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('order manage module uses explicit command session type', () => {
  const source = readFileSync(resolve(modulesDir, './orderManage.module.ts'), 'utf8')

  assert.doesNotMatch(source, /readonly session: any/)
  assert.match(source, /import type \{ Command, Context, Session \} from 'koishi'/)
  assert.match(source, /readonly session: Session/)
})
