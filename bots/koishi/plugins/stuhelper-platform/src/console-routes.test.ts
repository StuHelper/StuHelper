import assert from 'node:assert/strict'
import test from 'node:test'

import { Context } from 'koishi'

import { STUHELPER_PLATFORM_SERVICE } from './constants'
import {
  StuhelperPlatformDataService,
  registerPlatformConsoleRoutes,
} from './console-routes'

type Listener = (input?: unknown) => Promise<unknown> | unknown
type SavedConfig = Record<string, unknown>

interface EnabledChange {
  readonly moduleId: string
  readonly enabled: boolean
  readonly actor: string
}

interface ConfigSave {
  readonly moduleId: string
  readonly config: SavedConfig
  readonly actor: string
}

class FakeConsole {
  readonly listeners = new Map<string, Listener>()
  readonly authorities = new Map<string, number | undefined>()
  readonly refreshed: string[] = []

  addListener(name: string, listener: Listener, options?: { readonly authority?: number }) {
    this.listeners.set(name, listener)
    this.authorities.set(name, options?.authority)
  }

  async refresh(key: string) {
    this.refreshed.push(key)
  }
}

class FakePlatformService {
  readonly enabledChanges: EnabledChange[] = []
  readonly configSaves: ConfigSave[] = []
  readonly auditLimits: Array<number | undefined> = []

  readonly modules = [{
    manifest: { id: 'warn', name: 'Warn', order: 10 },
    enabled: true,
    config: { threshold: 3 },
  }]

  readonly auditEvents = [{
    id: 'audit-1',
    actor: 'console',
    moduleId: 'warn',
    action: 'module.state.set',
    summary: 'changed',
    payload: { enabled: false },
    createdAt: '2026-04-24T10:00:00.000Z',
    updatedAt: '2026-04-24T10:00:00.000Z',
  }]

  async listModules() {
    return this.modules
  }

  async listAuditEvents(limit?: number) {
    this.auditLimits.push(limit)
    return this.auditEvents
  }

  async setModuleEnabled(moduleId: string, enabled: boolean, actor: string) {
    this.enabledChanges.push({ moduleId, enabled, actor })
  }

  async saveModuleConfig(moduleId: string, config: SavedConfig, actor: string) {
    this.configSaves.push({ moduleId, config, actor })
  }
}

test('platform data service exposes modules and audit events', async () => {
  const service = new FakePlatformService()
  const dataService = new StuhelperPlatformDataService(new Context(), { service })

  const data = await dataService.get()

  assert.match(data.generatedAt, /^\d{4}-\d{2}-\d{2}T/)
  assert.deepEqual(data.modules, service.modules)
  assert.deepEqual(data.auditEvents, service.auditEvents)
  assert.deepEqual(service.auditLimits, [50])
})

test('registerPlatformConsoleRoutes installs stable console listeners', () => {
  const console = new FakeConsole()

  registerPlatformConsoleRoutes(createConsoleContext(console), new FakePlatformService())

  assert.deepEqual([...console.listeners.keys()], [
    'stuhelper-platform/refresh',
    'stuhelper-platform/module.set-enabled',
    'stuhelper-platform/module.save-config',
    'stuhelper-platform/audit.list',
  ])
  assert.equal(console.authorities.get('stuhelper-platform/refresh'), 4)
})

test('set-enabled listener validates input delegates and refreshes data', async () => {
  const console = new FakeConsole()
  const service = new FakePlatformService()
  registerPlatformConsoleRoutes(createConsoleContext(console), service)

  await callListener(console, 'stuhelper-platform/module.set-enabled', {
    moduleId: 'warn',
    enabled: false,
  })

  assert.deepEqual(service.enabledChanges, [{
    moduleId: 'warn',
    enabled: false,
    actor: 'console',
  }])
  assert.deepEqual(console.refreshed, [STUHELPER_PLATFORM_SERVICE])
})

test('set-enabled listener rejects invalid input before service call', async () => {
  const console = new FakeConsole()
  const service = new FakePlatformService()
  registerPlatformConsoleRoutes(createConsoleContext(console), service)

  await assert.rejects(
    callListener(console, 'stuhelper-platform/module.set-enabled', {
      moduleId: 'warn',
      enabled: 'no',
    }),
    /enabled must be a boolean/,
  )
  assert.deepEqual(service.enabledChanges, [])
  assert.deepEqual(console.refreshed, [])
})

test('save-config listener rejects invalid config before service call', async () => {
  const console = new FakeConsole()
  const service = new FakePlatformService()
  registerPlatformConsoleRoutes(createConsoleContext(console), service)

  await assert.rejects(
    callListener(console, 'stuhelper-platform/module.save-config', {
      moduleId: 'warn',
      config: null,
    }),
    /config must be an object/,
  )
  assert.deepEqual(service.configSaves, [])
  assert.deepEqual(console.refreshed, [])
})

test('save-config listener delegates valid config and refreshes data', async () => {
  const console = new FakeConsole()
  const service = new FakePlatformService()
  registerPlatformConsoleRoutes(createConsoleContext(console), service)

  await callListener(console, 'stuhelper-platform/module.save-config', {
    moduleId: 'warn',
    config: { threshold: 5 },
  })

  assert.deepEqual(service.configSaves, [{
    moduleId: 'warn',
    config: { threshold: 5 },
    actor: 'console',
  }])
  assert.deepEqual(console.refreshed, [STUHELPER_PLATFORM_SERVICE])
})

test('audit list listener validates optional limit and delegates to service', async () => {
  const console = new FakeConsole()
  const service = new FakePlatformService()
  registerPlatformConsoleRoutes(createConsoleContext(console), service)

  const result = await callListener(console, 'stuhelper-platform/audit.list', { limit: 2 })

  assert.deepEqual(result, service.auditEvents)
  assert.deepEqual(service.auditLimits, [2])
})

test('audit list listener rejects invalid limit before service call', async () => {
  const console = new FakeConsole()
  const service = new FakePlatformService()
  registerPlatformConsoleRoutes(createConsoleContext(console), service)

  await assert.rejects(
    callListener(console, 'stuhelper-platform/audit.list', { limit: 0 }),
    /limit must be a positive integer/,
  )
  assert.deepEqual(service.auditLimits, [])
})

function createConsoleContext(console: FakeConsole): Context {
  return { console } as unknown as Context
}

async function callListener(console: FakeConsole, name: string, input?: unknown) {
  const listener = console.listeners.get(name)
  assert.ok(listener, `missing listener: ${name}`)
  return listener(input)
}
