import assert from 'node:assert/strict'
import test from 'node:test'

import type { StuhelperModule } from './module-contract'
import { createModuleRegistry } from './module-registry'
import { StuhelperPlatformService } from './platform-service'
import type { AuditEventRecord } from './platform-models'

const DEFAULT_THRESHOLD = 3
const CUSTOM_THRESHOLD = 7
const DEFAULT_ORDER = 100

type Config = Record<string, unknown>

type FrozenTestConfig = Config & { readonly threshold: number; readonly nested: { readonly enabled: boolean }; readonly items: readonly string[] }
type MutableFrozenTestConfig = Config & { threshold: number }
type EnabledChange = { readonly moduleId: string; readonly enabled: boolean; readonly actor: string }
type ConfigSave = { readonly moduleId: string; readonly config: Config; readonly actor: string }

class FakeStore {
  readonly configs = new Map<string, Config>()
  readonly states = new Map<string, boolean>()
  readonly stateRequests: Array<{ moduleId: string; defaultEnabled: boolean }> = []
  readonly enabledChanges: EnabledChange[] = []
  readonly configSaves: ConfigSave[] = []
  readonly auditLimits: Array<number | undefined> = []
  readonly auditEvents: AuditEventRecord[] = []

  async getModuleState(moduleId: string, defaultEnabled: boolean) {
    this.stateRequests.push({ moduleId, defaultEnabled })
    const enabled = this.states.get(moduleId) ?? defaultEnabled
    return { enabled, status: enabled ? 'loaded' as const : 'disabled' as const, lastError: null }
  }

  async getModuleConfig(moduleId: string): Promise<Config | null> {
    return this.configs.get(moduleId) ?? null
  }

  async listAuditEvents(limit?: number): Promise<readonly AuditEventRecord[]> {
    this.auditLimits.push(limit)
    return limit === undefined ? this.auditEvents : this.auditEvents.slice(0, limit)
  }

  async setModuleEnabled(moduleId: string, enabled: boolean, actor: string) {
    this.enabledChanges.push({ moduleId, enabled, actor })
  }

  async saveModuleConfig(moduleId: string, config: Config, actor: string) {
    this.configSaves.push({ moduleId, config, actor })
  }
}

class FakeRuntime {
  readonly syncedModules: string[] = []
  readonly restartedModules: string[] = []

  async syncModule(moduleId: string): Promise<void> {
    this.syncedModules.push(moduleId)
  }

  async restartModule(moduleId: string): Promise<void> {
    this.restartedModules.push(moduleId)
  }
}

test('listModules returns registry snapshots with state config and metadata', async () => {
  const warn = moduleOf({ id: 'warn', order: 20, defaultEnabled: true })
  const audit = moduleOf({ id: 'audit', order: 10, defaultEnabled: false })
  const store = new FakeStore()
  store.configs.set('warn', { threshold: CUSTOM_THRESHOLD })
  store.states.set('warn', false)

  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([warn, audit]),
    store,
    runtime: new FakeRuntime(),
  })

  const snapshots = await service.listModules()

  assert.deepEqual(snapshots.map((snapshot) => snapshot.manifest.id), ['audit', 'warn'])
  assert.equal(snapshots[0].enabled, false)
  assert.equal(snapshots[0].status, 'disabled')
  assert.deepEqual(snapshots[0].config, { threshold: DEFAULT_THRESHOLD })
  assert.equal(snapshots[1].enabled, false)
  assert.equal(snapshots[1].lastError, null)
  assert.deepEqual(snapshots[1].config, { threshold: CUSTOM_THRESHOLD })
  assert.deepEqual(snapshots[1].permissions, warn.permissions)
  assert.deepEqual(snapshots[1].commands, [{
    name: 'warn',
    description: '测试命令',
    permission: 'warn.manage',
  }])
  assert.equal('register' in snapshots[1].commands[0], false)
  assert.deepEqual(snapshots[1].events, [{ name: 'warn.event' }])
  assert.deepEqual(snapshots[1].webui, warn.webui)
  assert.deepEqual(store.stateRequests, [
    { moduleId: 'audit', defaultEnabled: false },
    { moduleId: 'warn', defaultEnabled: true },
  ])
})

test('listModules returns immutable arrays detached from module internals', async () => {
  const module = moduleOf({ id: 'warn' })
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([module]),
    store: new FakeStore(),
    runtime: new FakeRuntime(),
  })

  const first = await service.listModules()
  const second = await service.listModules()

  assert.notEqual(first, second)
  assert.notEqual(first[0].permissions, module.permissions)
  assert.notEqual(first[0].commands, module.commands)
  assert.notEqual(first[0].events, module.events)
  assert.notEqual(first[0].webui, module.webui)
  assert.equal(Object.isFrozen(first), true)
  assert.equal(Object.isFrozen(first[0].permissions), true)
})

test('listModules returns deeply frozen config snapshots', async () => {
  const store = new FakeStore()
  store.configs.set('warn', {
    threshold: CUSTOM_THRESHOLD,
    nested: { enabled: true },
    items: ['existing'],
  })
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([moduleOf({ id: 'warn' })]),
    store,
    runtime: new FakeRuntime(),
  })

  const [snapshot] = await service.listModules()
  const config = snapshot.config as FrozenTestConfig

  assert.deepEqual(config, {
    threshold: CUSTOM_THRESHOLD,
    nested: { enabled: true },
    items: ['existing'],
  })
  assert.equal(Object.isFrozen(config), true)
  assert.equal(Object.isFrozen(config.nested), true)
  assert.equal(Object.isFrozen(config.items), true)
  attemptFrozenMutation(() => {
    ;(config as MutableFrozenTestConfig).threshold = DEFAULT_THRESHOLD
  })
  attemptFrozenMutation(() => {
    ;(config.nested as { enabled: boolean }).enabled = false
  })
  attemptFrozenMutation(() => {
    ;(config.items as string[]).push('extra')
  })
  assert.equal(config.threshold, CUSTOM_THRESHOLD)
  assert.equal(config.nested.enabled, true)
  assert.deepEqual(config.items, ['existing'])
})

test('listModules validates stored config through module schema', async () => {
  const store = new FakeStore()
  store.configs.set('warn', { threshold: 'bad' })
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([moduleOf({ id: 'warn' })]),
    store,
    runtime: new FakeRuntime(),
  })

  await assert.rejects(
    () => service.listModules(),
    /threshold must be a number/,
  )
})

test('setModuleEnabled rejects unknown modules and delegates known modules', async () => {
  const store = new FakeStore()
  const runtime = new FakeRuntime()
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([moduleOf({ id: 'warn' })]),
    store,
    runtime,
  })

  await assert.rejects(
    () => service.setModuleEnabled('missing', true, 'admin'),
    /unknown StuHelper module id: missing/,
  )
  assert.deepEqual(store.enabledChanges, [])

  await service.setModuleEnabled('warn', false, 'admin')

  assert.deepEqual(store.enabledChanges, [
    { moduleId: 'warn', enabled: false, actor: 'admin' },
  ])
  assert.deepEqual(runtime.syncedModules, ['warn'])
})

test('saveModuleConfig validates config before saving parsed value', async () => {
  const store = new FakeStore()
  const runtime = new FakeRuntime()
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([moduleOf({ id: 'warn' })]),
    store,
    runtime,
  })

  await assert.rejects(
    () => service.saveModuleConfig('missing', { threshold: CUSTOM_THRESHOLD }, 'admin'),
    /unknown StuHelper module id: missing/,
  )
  await assert.rejects(
    () => service.saveModuleConfig('warn', { threshold: 'bad' }, 'admin'),
    /threshold must be a number/,
  )
  assert.deepEqual(store.configSaves, [])

  await service.saveModuleConfig('warn', {
    threshold: CUSTOM_THRESHOLD,
    discarded: true,
  }, 'admin')

  assert.deepEqual(store.configSaves, [
    {
      moduleId: 'warn',
      config: { threshold: CUSTOM_THRESHOLD },
      actor: 'admin',
    },
  ])
  assert.deepEqual(runtime.restartedModules, ['warn'])
})

function moduleOf(input: {
  readonly id: string
  readonly order?: number
  readonly defaultEnabled?: boolean
}): StuhelperModule<Config> {
  return {
    manifest: {
      id: input.id,
      name: input.id,
      description: `${input.id} module`,
      version: '0.0.0',
      category: 'system',
      defaultEnabled: input.defaultEnabled ?? true,
      order: input.order ?? DEFAULT_ORDER,
    },
    configSchema: {
      parse: parseConfig,
      defaults: () => ({ threshold: DEFAULT_THRESHOLD }),
    },
    permissions: [{
      id: `${input.id}.manage`,
      label: '管理',
      description: '管理模块',
    }],
    commands: [{
      name: input.id,
      description: '测试命令',
      permission: `${input.id}.manage`,
    }],
    events: [{
      name: `${input.id}.event`,
    }],
    webui: [{
      id: `${input.id}.settings`,
      label: '设置',
      section: 'module',
    }],
    setup: () => {},
  }
}

function parseConfig(value: unknown): Config {
  const config = value as Config
  if (typeof config.threshold !== 'number') {
    throw new Error('threshold must be a number')
  }
  const parsed: Config = { threshold: config.threshold }
  if (config.nested && typeof config.nested === 'object') parsed.nested = config.nested
  if (Array.isArray(config.items)) parsed.items = config.items
  return parsed
}

function attemptFrozenMutation(action: () => void): void {
  try {
    action()
  } catch (error) {
    assert.equal(error instanceof TypeError, true)
  }
}
