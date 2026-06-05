import assert from 'node:assert/strict'
import test from 'node:test'

import type { WarnModule } from './warn.module'
import { migrateWarnData } from './warn-migration'

test('migrateWarnData migrates legacy numeric and composite warn records safely', () => {
  const harness = createWarnMigrationHarness({
    'guild-1': {
      user1: 2,
      user2: { count: 3, timestamp: 10 },
      malformed: { count: 'bad', timestamp: 11 },
    },
    'guild-2:user3': {
      groups: {
        'guild-2': { count: 4, timestamp: 20 },
      },
    },
    'guild-3:user4': {
      groups: {
        other: { count: 5, timestamp: 30 },
      },
    },
  })

  migrateWarnData(harness.host)

  assert.deepEqual(harness.deletes, ['guild-1', 'guild-2:user3'])
  assert.equal(harness.records['guild-2:user3'], undefined)
  assert.deepEqual(harness.records['guild-3:user4'], {
    groups: {
      other: { count: 5, timestamp: 30 },
    },
  })

  const guild1 = harness.records['guild-1'] as Record<string, { count: number; timestamp: number }>
  assert.equal(guild1.user1.count, 2)
  assert.equal(typeof guild1.user1.timestamp, 'number')
  assert.deepEqual(guild1.user2, { count: 3, timestamp: 10 })
  assert.equal(guild1.malformed, undefined)

  const guild2 = harness.records['guild-2'] as Record<string, { count: number; timestamp: number }>
  assert.deepEqual(guild2.user3, { count: 4, timestamp: 20 })
  assert.equal(harness.flushes, 1)
  assert.deepEqual(harness.logs, ['警告数据已迁移到新格式'])
})

test('migrateWarnData ignores malformed warn stores without flushing', () => {
  const harness = createWarnMigrationHarness({
    scalar: 1,
    missing: null,
    invalid: {
      user1: { count: 'bad' },
      user2: { timestamp: 20 },
    },
  })

  migrateWarnData(harness.host)

  assert.equal(harness.flushes, 0)
  assert.deepEqual(harness.deletes, [])
  assert.deepEqual(harness.logs, [])
})

function createWarnMigrationHarness(initialRecords: Record<string, unknown>) {
  const records = { ...initialRecords }
  const deletes: string[] = []
  const logs: string[] = []
  let flushes = 0

  const host = {
    data: {
      warns: {
        getAll: () => records,
        delete(key: string) {
          deletes.push(key)
          delete records[key]
        },
        set(key: string, value: unknown) {
          records[key] = value
        },
        flush() {
          flushes += 1
        },
      },
    },
    ctx: {
      logger: () => ({
        info(message: string) {
          logs.push(message)
        },
      }),
    },
  } as unknown as WarnModule

  return {
    host,
    records,
    deletes,
    logs,
    get flushes() {
      return flushes
    },
  }
}
