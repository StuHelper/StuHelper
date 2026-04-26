import assert from 'node:assert/strict'
import test from 'node:test'

import { Context } from 'koishi'

import { BaseModule } from '../core/modules'
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
  ['manage-message', {}, { setEssenceMsg: { enabled: true, authority: 3 } }],
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
] as const

for (const [id, data, config] of nativeRuntimeCases) {
  test(`${id} runtime module is native instead of adapted BaseModule`, () => {
    const module = getRuntimeModules().find(item => item.id === id)
    assert.ok(module)

    const instance = module.create(new Context(), {
      service: {} as any,
      data: data as any,
      config: config as any,
    })

    assert.equal(instance.meta.name, id)
    assert.ok(!(instance instanceof BaseModule))
  })
}
