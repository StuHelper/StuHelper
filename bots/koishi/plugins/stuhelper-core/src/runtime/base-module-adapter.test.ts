import assert from 'node:assert/strict'
import test from 'node:test'

import { Context } from 'koishi'

import { BaseModule, type ModuleMeta } from '../core/modules'
import { adaptBaseModule } from './base-module-adapter'

class FakeModule extends BaseModule {
  static initCalls = 0
  static disposeCalls = 0

  readonly meta: ModuleMeta = {
    name: 'fake',
    description: 'Fake module',
  }

  getDeps() {
    return {
      ctx: this.ctx,
      data: this.data,
      config: this.config,
    }
  }

  protected async onInit() {
    FakeModule.initCalls += 1
  }

  protected async onDispose() {
    FakeModule.disposeCalls += 1
  }
}

test('adaptBaseModule creates runtime module instances with injected deps', async () => {
  const ctx = new Context()
  const data = { dataPath: '/tmp/stuhelper-test' } as any
  const config = { keywords: [] } as any
  const module = adaptBaseModule({
    id: 'fake',
    order: 7,
    ModuleType: FakeModule,
  })

  const instance = module.create(ctx, { data, config, service: {} as any })

  assert.equal(module.id, 'fake')
  assert.equal(module.order, 7)
  assert.ok(instance instanceof FakeModule)
  assert.deepEqual(instance.getDeps(), { ctx, data, config })

  await instance.init()
  await instance.dispose()
  assert.equal(FakeModule.initCalls, 1)
  assert.equal(FakeModule.disposeCalls, 1)
})
