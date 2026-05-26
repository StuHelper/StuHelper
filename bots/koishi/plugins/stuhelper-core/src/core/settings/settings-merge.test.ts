import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { deepMerge, getDiff } from './settings-merge'

interface SampleSettings {
  enabled: boolean
  nested: {
    count: number
    label: string
  }
  flags: string[]
  optional?: string
}

const defaults: SampleSettings = {
  enabled: true,
  nested: {
    count: 1,
    label: 'default',
  },
  flags: ['alpha'],
}

test('deepMerge applies nested overrides without mutating defaults', () => {
  const merged = deepMerge(defaults, {
    enabled: false,
    nested: {
      count: 2,
    },
    flags: ['beta'],
    optional: undefined,
  })

  assert.deepEqual(merged, {
    enabled: false,
    nested: {
      count: 2,
      label: 'default',
    },
    flags: ['beta'],
  })
  assert.deepEqual(defaults, {
    enabled: true,
    nested: {
      count: 1,
      label: 'default',
    },
    flags: ['alpha'],
  })
})

test('getDiff returns only values that differ from defaults', () => {
  const current: SampleSettings = {
    enabled: true,
    nested: {
      count: 3,
      label: 'default',
    },
    flags: ['alpha'],
    optional: 'custom',
  }

  assert.deepEqual(getDiff(defaults, current), {
    nested: {
      count: 3,
    },
    optional: 'custom',
  })
})

test('settings merge helpers keep unknown boundaries instead of any', () => {
  const sourcePath = resolve(dirname(fileURLToPath(import.meta.url)), 'settings-merge.ts')
  const source = readFileSync(sourcePath, 'utf8')

  assert.doesNotMatch(source, /\bany\b/)
  assert.match(source, /type PlainRecord = Record<string, unknown>/)
  assert.match(source, /function arraysEqual\(a: readonly unknown\[\], b: unknown\): boolean/)
})
