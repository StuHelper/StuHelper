import assert from 'node:assert/strict'
import test from 'node:test'

import { Context } from 'koishi'

import { getRuntimeModules } from './registry'

test('runtime registry keeps P3 module order', () => {
  const ids = getRuntimeModules().map(module => module.id)

  assert.deepEqual(ids, [
    'warn',
    'keyword',
    'manage-member',
    'manage-message',
    'manage-order',
    'antirepeat',
    'welcome',
    'repeat',
    'dice',
    'banme',
    'antirecall',
    'ai',
    'config',
    'log',
    'subscription',
    'help',
    'report',
    'getauth',
    'auth',
    'event',
    'status',
    'manage-cross-group',
  ])
})

const nativeRuntimeCases = [
  ['help', {}, {}],
  ['dice', { groupConfig: {} }, {}],
  ['banme', {}, { banme: {} }],
  ['config', {}, {}],
  ['status', {}, {}],
  ['event', {}, {}],
  ['antirepeat', { groupConfig: {} }, { antiRepeat: { threshold: 5 } }],
  ['manage-message', {}, {}],
  ['getauth', {}, {}],
  ['manage-cross-group', {}, {}],
  ['auth', {}, {}],
  ['repeat', { groupConfig: { get: () => undefined } }, {}],
  ['welcome', {}, {}],
  ['log', { dataPath: process.cwd() }, {}],
  ['antirecall', {}, {}],
  ['subscription', {}, {}],
  ['manage-member', {}, {}],
  ['manage-order', {}, {}],
  ['keyword', {}, {}],
  ['ai', { dataPath: process.cwd() }, {}],
  ['warn', {}, {}],
  ['report', {}, {}],
] as const

for (const [id, data, config] of nativeRuntimeCases) {
  test(`${id} runtime module creates a runtime instance`, () => {
    const module = getRuntimeModules().find(item => item.id === id)
    assert.ok(module)

    const instance = module.create(new Context(), {
      service: {} as any,
      data: data as any,
      config: config as any,
    })

    assert.equal(instance.meta.name, id)
    assert.equal(instance.state, 'unloaded')
    assert.equal(typeof instance.init, 'function')
    assert.equal(typeof instance.dispose, 'function')
  })
}

const configDependentRuntimeCases = [
  'warn',
  'keyword',
  'manage-member',
  'manage-order',
  'antirepeat',
  'welcome',
  'banme',
  'antirecall',
  'ai',
  'config',
  'log',
  'subscription',
  'report',
] as const

test('runtime module config getters require the live stuhelperGroupCenter service', () => {
  for (const id of configDependentRuntimeCases) {
    const module = getRuntimeModules().find(item => item.id === id)
    assert.ok(module, `${id} module should be registered`)

    const instance = module.create(new Context(), {
      service: {} as any,
      data: createDataForRuntimeModule(id),
      config: { marker: id } as any,
    }) as any

    assert.throws(
      () => instance.config,
      /stuhelperGroupCenter service is required/,
      `${id} config getter should not fall back to constructor config`,
    )
  }
})

function createDataForRuntimeModule(id: string): any {
  if (id === 'log' || id === 'ai') {
    return { dataPath: process.cwd() }
  }
  if (id === 'antirepeat') {
    return { groupConfig: {} }
  }
  return {}
}
