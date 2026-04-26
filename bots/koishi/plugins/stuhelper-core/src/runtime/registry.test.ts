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

test('antirepeat runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'antirepeat')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: { groupConfig: {} } as any,
    config: { antiRepeat: { threshold: 5 } } as any,
  })

  assert.equal(instance.meta.name, 'antirepeat')
  assert.ok(!(instance instanceof BaseModule))
})

test('message manage runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'manage-message')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: { setEssenceMsg: { enabled: true, authority: 3 } } as any,
  })

  assert.equal(instance.meta.name, 'manage-message')
  assert.ok(!(instance instanceof BaseModule))
})

test('getauth runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'getauth')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'getauth')
  assert.ok(!(instance instanceof BaseModule))
})

test('cross group runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'manage-cross-group')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'manage-cross-group')
  assert.ok(!(instance instanceof BaseModule))
})

test('auth runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'auth')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'auth')
  assert.ok(!(instance instanceof BaseModule))
})

test('repeat runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'repeat')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: { groupConfig: { get: () => undefined } } as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'repeat')
  assert.ok(!(instance instanceof BaseModule))
})

test('welcome runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'welcome')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: {} as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'welcome')
  assert.ok(!(instance instanceof BaseModule))
})

test('log runtime module is native instead of adapted BaseModule', () => {
  const module = getRuntimeModules().find(item => item.id === 'log')
  assert.ok(module)

  const instance = module.create(new Context(), {
    service: {} as any,
    data: { dataPath: process.cwd() } as any,
    config: {} as any,
  })

  assert.equal(instance.meta.name, 'log')
  assert.ok(!(instance instanceof BaseModule))
})
