import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { DataManager } from './data.service'

test('DataManager stores JSON files under STUHELPER_GROUP_CENTER_DATA_DIR when set', (t) => {
  const previous = process.env.STUHELPER_GROUP_CENTER_DATA_DIR
  const baseDir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-data-base-'))
  const dataDir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-data-env-'))

  process.env.STUHELPER_GROUP_CENTER_DATA_DIR = dataDir
  t.after(() => {
    if (previous === undefined) {
      delete process.env.STUHELPER_GROUP_CENTER_DATA_DIR
    } else {
      process.env.STUHELPER_GROUP_CENTER_DATA_DIR = previous
    }
  })

  const manager = new DataManager({
    baseDir,
    logger: () => ({
      info() {},
      warn() {},
      error() {},
      debug() {},
    }),
  } as any)
  t.after(() => manager.dispose())

  manager.warns.set('guild-1', { user1: { count: 1, timestamp: 1 } })
  manager.warns.flush()

  assert.equal(manager.dataPath, path.resolve(dataDir))
  assert.equal(fs.existsSync(path.join(dataDir, 'warns.json')), true)
  assert.equal(fs.existsSync(path.join(baseDir, 'data/stuhelperGroupCenter/warns.json')), false)
})
