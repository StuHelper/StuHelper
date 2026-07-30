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

test('StuhelperGroupCenterService exposes the canonical console guild scope resolver', async () => {
  const service = Object.create(StuhelperGroupCenterService.prototype) as any
  service._auth = {
    getRoles: () => [{ id: 'guild-operator', guildIds: ['guild-1'] }],
    getUserRoleIds: (userId: string) => userId === 'onebot:operator'
      ? ['guild-operator']
      : [],
  }
  service.context = {
    database: {
      async get(table: string, query: { aid: number }) {
        assert.equal(table, 'binding')
        assert.deepEqual(query, { aid: 42 })
        return [{ platform: 'onebot', pid: 'operator' }]
      },
    },
  }

  const scope = await service.resolveConsoleGuildScope({
    auth: {
      id: 42,
      authority: 4,
    },
  })

  assert.equal(scope.kind, 'guilds')
  assert.deepEqual([...scope.guildIds], ['guild-1'])
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
