import { strict as assert } from 'node:assert'
import { test } from 'node:test'

import { SettingsReadCache } from './settings-cache'

test('read 只加载一次并复用缓存', async () => {
  const cache = new SettingsReadCache<{ value: number }>()
  let loads = 0
  const load = async () => {
    loads += 1
    return { value: loads }
  }

  const first = await cache.read(load)
  const second = await cache.read(load)

  assert.equal(loads, 1)
  assert.equal(first, second)
})

test('并发首次读取共享同一次加载', async () => {
  const cache = new SettingsReadCache<{ value: number }>()
  let loads = 0
  const load = async () => {
    loads += 1
    return { value: loads }
  }

  const [first, second] = await Promise.all([cache.read(load), cache.read(load)])

  assert.equal(loads, 1)
  assert.equal(first, second)
})

test('加载失败不缓存，下一次读取重新加载', async () => {
  const cache = new SettingsReadCache<{ value: number }>()
  let loads = 0
  const load = async () => {
    loads += 1
    if (loads === 1) {
      throw new Error('db unavailable')
    }
    return { value: loads }
  }

  await assert.rejects(cache.read(load), /db unavailable/)
  const recovered = await cache.read(load)

  assert.equal(loads, 2)
  assert.equal(recovered.value, 2)
})

test('write 回填缓存，后续读取不再加载', async () => {
  const cache = new SettingsReadCache<{ value: number }>()
  let loads = 0
  const load = async () => {
    loads += 1
    return { value: 0 }
  }

  const written = cache.write({ value: 42 })
  const read = await cache.read(load)

  assert.equal(loads, 0)
  assert.equal(read, written)
  assert.equal(read.value, 42)
})

test('invalidate 后重新加载', async () => {
  const cache = new SettingsReadCache<{ value: number }>()
  let loads = 0
  const load = async () => {
    loads += 1
    return { value: loads }
  }

  await cache.read(load)
  cache.invalidate()
  const reloaded = await cache.read(load)

  assert.equal(loads, 2)
  assert.equal(reloaded.value, 2)
})

test('缓存值被深冻结，原地修改不生效', async () => {
  const cache = new SettingsReadCache<{ nested: { flag: boolean } }>()
  const value = await cache.read(async () => ({ nested: { flag: true } }))

  try {
    (value.nested as { flag: boolean }).flag = false
  } catch {
    // 严格模式下冻结写入抛 TypeError，非严格模式静默忽略；两种都接受
  }

  assert.equal(value.nested.flag, true)
})

test('settingsReadCacheFor：同库同表共享缓存，不同表/不同库隔离', async () => {
  const { settingsReadCacheFor } = await import('./settings-cache')
  const databaseA = { tables: {} }
  const databaseB = { tables: {} }

  const first = settingsReadCacheFor(databaseA, 'settings_table')
  const sameTable = settingsReadCacheFor(databaseA, 'settings_table')
  const otherTable = settingsReadCacheFor(databaseA, 'other_table')
  const otherDatabase = settingsReadCacheFor(databaseB, 'settings_table')

  assert.equal(first, sameTable)
  assert.notEqual(first, otherTable)
  assert.notEqual(first, otherDatabase)
})

test('settingsReadCacheFor：以 tables 为身份锚点，跨代理访问共享', async () => {
  const { settingsReadCacheFor } = await import('./settings-cache')
  const tables = {}
  const proxyAccessA = { tables, get: () => {} }
  const proxyAccessB = { tables, set: () => {} }

  assert.equal(
    settingsReadCacheFor(proxyAccessA, 'settings_table'),
    settingsReadCacheFor(proxyAccessB, 'settings_table'),
  )
})
