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

test('JsonDataStore delayed flush failure is logged and retried instead of crashing the timer', async (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-json-store-'))
  const filePath = path.join(dir, 'data.json')
  const errors: unknown[][] = []
  const store = new JsonDataStore(filePath, { value: 1 }, {
    createBackup: false,
    saveDelay: 5,
    logger: {
      info() {},
      error(...args: unknown[]) {
        errors.push(args)
      },
    },
  })
  const mocked = t.mock.method(fs, 'writeFileSync', () => {
    throw new Error('disk write failed')
  })

  store.set('value', 2)
  await new Promise((resolve) => setTimeout(resolve, 50))

  // timer 回调内的 flush 失败被捕获并记录，进程不会被未捕获异常击穿
  // （flush 自身记录"保存数据失败"，timer 捕获层再记录"延迟保存失败"）
  assert.equal(errors.length, 2)
  assert.match(String(errors[0][0]), /保存数据失败/)
  assert.match(String(errors[1][0]), /延迟保存失败/)

  // dirty 状态保留：写盘恢复后再次 flush 可成功落盘
  mocked.mock.restore()
  store.flush()
  assert.equal(JSON.parse(fs.readFileSync(filePath, 'utf8')).value, 2)
})

test('JsonDataStore escapes backup basename when cleaning old backups', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-json-store-'))
  const filePath = path.join(dir, 'data+1.json')
  const unrelatedBackup = path.join(dir, 'dataa1.backup.2020-01-01T00-00-00-000Z.json')
  const oldBackups = [
    path.join(dir, 'data+1.backup.2020-01-01T00-00-00-000Z.json'),
    path.join(dir, 'data+1.backup.2020-01-02T00-00-00-000Z.json'),
  ]

  fs.writeFileSync(filePath, '{"value":1}', 'utf8')
  fs.writeFileSync(unrelatedBackup, '{}', 'utf8')
  for (const backup of oldBackups) {
    fs.writeFileSync(backup, '{}', 'utf8')
    fs.utimesSync(backup, new Date('2020-01-01T00:00:00.000Z'), new Date('2020-01-01T00:00:00.000Z'))
  }

  const store = new JsonDataStore(filePath, { value: 1 }, { maxBackups: 1 })
  store.set('value', 2)
  store.flush()

  const files = fs.readdirSync(dir)

  assert.equal(files.filter(file => file.startsWith('data+1.backup.')).length, 1)
  assert.ok(fs.existsSync(unrelatedBackup))
})
