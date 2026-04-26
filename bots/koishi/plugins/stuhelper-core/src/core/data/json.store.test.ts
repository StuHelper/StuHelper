import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { JsonDataStore } from './json.store'

test('JsonDataStore throws when existing JSON data is corrupt', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-json-store-'))
  const filePath = path.join(dir, 'data.json')
  fs.writeFileSync(filePath, '{ invalid json', 'utf8')

  assert.throws(
    () => new JsonDataStore(filePath, { list: [] }),
    SyntaxError,
  )
})

test('JsonDataStore throws when saving data fails', (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-json-store-'))
  const filePath = path.join(dir, 'data.json')
  const store = new JsonDataStore(filePath, { value: 1 }, { createBackup: false })
  const writeError = new Error('disk write failed')

  store.set('value', 2)
  t.mock.method(fs, 'writeFileSync', () => {
    throw writeError
  })

  assert.throws(() => store.flush(), /disk write failed/)
})
