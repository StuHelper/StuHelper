import assert from 'node:assert/strict'
import test from 'node:test'

import { createModuleRegistry } from './module-registry'
import type { StuhelperModule } from './module-contract'

function moduleOf(id: string, order = 100): StuhelperModule<Record<string, never>> {
  return {
    manifest: {
      id,
      name: id,
      description: `${id} module`,
      version: '0.0.0',
      category: 'system',
      defaultEnabled: true,
      order,
    },
    configSchema: {
      parse: (value) => value as Record<string, never>,
      defaults: () => ({}),
    },
    permissions: [],
    commands: [],
    events: [],
    webui: [],
    setup: () => {},
  }
}

test('registry sorts modules by order then id', () => {
  const registry = createModuleRegistry([
    moduleOf('review', 20),
    moduleOf('warn', 10),
    moduleOf('audit', 20),
  ])

  assert.deepEqual(registry.list().map((module) => module.manifest.id), [
    'warn',
    'audit',
    'review',
  ])
})

test('registry rejects duplicate module ids', () => {
  assert.throws(
    () => createModuleRegistry([moduleOf('warn'), moduleOf('warn')]),
    /duplicate StuHelper module id: warn/,
  )
})

test('registry gets modules by id', () => {
  const warn = moduleOf('warn')
  const registry = createModuleRegistry([warn])

  assert.equal(registry.get('warn'), warn)
  assert.equal(registry.get('missing'), null)
})

test('registry list cannot be polluted by callers', () => {
  const warn = moduleOf('warn')
  const registry = createModuleRegistry([warn])
  const listed = registry.list() as unknown as StuhelperModule[]

  assert.equal(listed.pop(), warn)
  assert.deepEqual(registry.list().map((module) => module.manifest.id), ['warn'])
  assert.equal(registry.get('warn'), warn)
})
