import assert from 'node:assert/strict'
import test from 'node:test'

import type { StuhelperModule } from './module-contract'
import { createModuleRegistry } from './module-registry'
import {
  StuhelperPlatformRuntime,
  type ModuleAuditInput,
  type ModuleStatusInput,
} from './platform-runtime'

const DEFAULT_THRESHOLD = 3
const CUSTOM_THRESHOLD = 7

type Config = Record<string, unknown>

class FakeKoishi {
  readonly scopes: Array<{ disposed: boolean }> = []

  plugin(callback: (ctx: unknown) => void) {
    const scope = { disposed: false }
    this.scopes.push(scope)
    callback({ scope })
    return {
      dispose: () => {
        scope.disposed = true
        return true
      },
    }
  }
}

class FakeStore {
  readonly configs = new Map<string, Config>()
  readonly states = new Map<string, boolean>()
  readonly loaded: ModuleStatusInput[] = []
  readonly errors: ModuleStatusInput[] = []
  readonly audits: ModuleAuditInput[] = []

  async getModuleState(moduleId: string, defaultEnabled: boolean) {
    return { enabled: this.states.get(moduleId) ?? defaultEnabled }
  }

  async getModuleConfig(moduleId: string): Promise<Config | null> {
    return this.configs.get(moduleId) ?? null
  }

  async markModuleLoaded(input: ModuleStatusInput): Promise<void> {
    this.loaded.push(input)
  }

  async markModuleError(input: ModuleStatusInput): Promise<void> {
    this.errors.push(input)
  }

  async appendAudit(input: ModuleAuditInput): Promise<void> {
    this.audits.push(input)
  }
}

test('runtime starts enabled modules with stored config and records loaded state', async () => {
  const store = new FakeStore()
  const koishi = new FakeKoishi()
  store.configs.set('guard', { threshold: CUSTOM_THRESHOLD })
  const module = moduleOf('guard')
  const runtime = new StuhelperPlatformRuntime({
    koishi: koishi as never,
    registry: createModuleRegistry([module]),
    store,
  })

  await runtime.start()

  assert.equal(koishi.scopes.length, 1)
  assert.deepEqual(module.seenConfigs, [{ threshold: CUSTOM_THRESHOLD }])
  assert.deepEqual(store.loaded, [statusOf('guard', 'loaded', null)])
  assert.deepEqual(store.audits, [{
    actor: 'system',
    moduleId: 'guard',
    action: 'module.setup',
    summary: 'loaded guard',
    payload: { threshold: CUSTOM_THRESHOLD },
  }])
})

test('runtime skips disabled modules without writing status', async () => {
  const store = new FakeStore()
  const koishi = new FakeKoishi()
  store.states.set('guard', false)
  const module = moduleOf('guard')
  const runtime = new StuhelperPlatformRuntime({
    koishi: koishi as never,
    registry: createModuleRegistry([module]),
    store,
  })

  await runtime.start()

  assert.equal(koishi.scopes.length, 0)
  assert.deepEqual(module.seenConfigs, [])
  assert.deepEqual(store.loaded, [])
  assert.deepEqual(store.errors, [])
})

test('runtime marks startup errors and rethrows the original failure', async () => {
  const store = new FakeStore()
  const koishi = new FakeKoishi()
  const module = moduleOf('guard', new Error('boom'))
  const runtime = new StuhelperPlatformRuntime({
    koishi: koishi as never,
    registry: createModuleRegistry([module]),
    store,
  })

  await assert.rejects(() => runtime.start(), /boom/)

  assert.deepEqual(store.loaded, [])
  assert.deepEqual(store.errors, [statusOf('guard', 'error', 'boom')])
  assert.equal(koishi.scopes[0].disposed, true)
})

test('runtime disables and restarts active module scopes', async () => {
  const store = new FakeStore()
  const koishi = new FakeKoishi()
  const module = moduleOf('guard')
  const runtime = new StuhelperPlatformRuntime({
    koishi: koishi as never,
    registry: createModuleRegistry([module]),
    store,
  })

  await runtime.syncModule('guard')
  store.states.set('guard', false)
  await runtime.syncModule('guard')
  store.states.set('guard', true)
  store.configs.set('guard', { threshold: CUSTOM_THRESHOLD })
  await runtime.restartModule('guard')

  assert.deepEqual(module.seenConfigs, [
    { threshold: DEFAULT_THRESHOLD },
    { threshold: CUSTOM_THRESHOLD },
  ])
  assert.equal(koishi.scopes[0].disposed, true)
  assert.equal(koishi.scopes[1].disposed, false)
})

function moduleOf(id: string, failure?: Error): StuhelperModule<Config> & {
  readonly seenConfigs: Config[]
} {
  const seenConfigs: Config[] = []
  return {
    manifest: {
      id,
      name: id,
      description: `${id} module`,
      version: '1.0.0',
      category: 'moderation',
      defaultEnabled: true,
      order: 10,
    },
    configSchema: {
      defaults: () => ({ threshold: DEFAULT_THRESHOLD }),
      parse: parseConfig,
    },
    permissions: [],
    commands: [],
    events: [],
    webui: [],
    seenConfigs,
    setup: async (context) => {
      if (failure) throw failure
      seenConfigs.push(context.config)
      await context.audit({
        moduleId: id,
        action: 'module.setup',
        summary: `loaded ${id}`,
        payload: context.config,
      })
    },
  }
}

function parseConfig(value: unknown): Config {
  const config = value as Config
  if (typeof config.threshold !== 'number') {
    throw new Error('threshold must be a number')
  }
  return { threshold: config.threshold }
}

function statusOf(
  moduleId: string,
  status: 'loaded' | 'error',
  lastError: string | null,
): ModuleStatusInput {
  return { moduleId, version: '1.0.0', order: 10, status, lastError }
}
