import assert from 'node:assert/strict'
import test from 'node:test'

import { StuhelperGroupCenterService } from './stuhelper-group-center.service'

test('StuhelperGroupCenterService propagates runtime module init failure', async () => {
  const calls: string[] = []
  const service = Object.create(StuhelperGroupCenterService.prototype) as any
  service._modules = new Map([
    ['broken', createRuntimeModule('broken', async () => {
      calls.push('broken')
      throw new Error('module init failed')
    })],
    ['later', createRuntimeModule('later', async () => {
      calls.push('later')
    })],
  ])
  service.serviceLogger = createLogger()
  service.warmCacheAsync = () => undefined

  await assert.rejects(() => service.initModules(), /module init failed/)
  assert.deepEqual(calls, ['broken'])
})

function createRuntimeModule(name: string, init: () => Promise<void>) {
  return {
    meta: { name },
    init,
  }
}

function createLogger() {
  return {
    error() {},
    info() {},
    warn() {},
  }
}
