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

test('help runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'help')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'help')
  assert.ok(!(instance instanceof BaseModule))
})

test('dice runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'dice')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: { groupConfig: {} } as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'dice')
  assert.ok(!(instance instanceof BaseModule))
})

test('banme runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'banme')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: { banme: {} } as any,
  })

  assert.equal(instance.meta.name, 'banme')
  assert.ok(!(instance instanceof BaseModule))
})

test('config runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'config')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'config')
  assert.ok(!(instance instanceof BaseModule))
})

test('status runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'status')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'status')
  assert.ok(!(instance instanceof BaseModule))
})

test('event runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'event')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'event')
  assert.ok(!(instance instanceof BaseModule))
})
