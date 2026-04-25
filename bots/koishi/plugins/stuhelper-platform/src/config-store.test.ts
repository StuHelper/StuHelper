import assert from 'node:assert/strict'
import test from 'node:test'

import {
  AUDIT_EVENT_TABLE,
  MODULE_CONFIG_TABLE,
  MODULE_STATE_TABLE,
} from './constants'
import { PlatformConfigStore } from './config-store'

type Row = Record<string, unknown>

const DEFAULT_MODULE_ORDER = 100
const TEST_THRESHOLD = 3
const EXISTING_SCHEMA_VERSION = 'stored-v2'
const NEWER_AUDIT_TIME = new Date('2026-04-24T10:00:00.000Z')
const OLDER_AUDIT_TIME = new Date('2026-04-24T09:00:00.000Z')

class FakeDatabase {
  readonly rows = new Map<string, Row[]>()

  async get(table: string, query: Row): Promise<Row[]> {
    return this.tableRows(table).filter((row) => matchesQuery(row, query))
  }

  async upsert(table: string, input: Row | readonly Row[]): Promise<void> {
    const nextRows = Array.isArray(input) ? input : [input]
    for (const row of nextRows) {
      this.upsertRow(table, row)
    }
  }

  async create(table: string, row: Row): Promise<void> {
    this.tableRows(table).push(row)
  }

  tableRows(table: string): Row[] {
    const rows = this.rows.get(table) ?? []
    this.rows.set(table, rows)
    return rows
  }

  private upsertRow(table: string, row: Row): void {
    const rows = this.tableRows(table)
    const index = rows.findIndex((current) => current.id === row.id)
    if (index === -1) {
      rows.push(row)
      return
    }
    rows[index] = { ...rows[index], ...row }
  }
}

test('config store returns default enabled state when no row exists', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })

  const state = await store.getModuleState('warn', true)

  assert.equal(state.id, 'warn')
  assert.equal(state.moduleId, 'warn')
  assert.equal(state.enabled, true)
  assert.equal(state.order, DEFAULT_MODULE_ORDER)
  assert.equal(state.version, '0.0.0')
  assert.equal(state.status, 'pending')
  assert.equal(state.lastError, null)
  assert.equal(database.tableRows(MODULE_STATE_TABLE).length, 0)
})

test('config store returns pending disabled default state when no row exists', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })

  const state = await store.getModuleState('warn', false)

  assert.equal(state.id, 'warn')
  assert.equal(state.moduleId, 'warn')
  assert.equal(state.enabled, false)
  assert.equal(state.status, 'disabled')
  assert.equal(database.tableRows(MODULE_STATE_TABLE).length, 0)
})

test('getModuleConfig returns null when missing', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })

  const config = await store.getModuleConfig('warn')

  assert.equal(config, null)
})

test('getModuleConfig returns stored config when present', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })
  const storedConfig = { threshold: TEST_THRESHOLD, mode: 'strict' }
  const now = new Date()

  database.tableRows(MODULE_CONFIG_TABLE).push({
    id: 'warn',
    moduleId: 'warn',
    schemaVersion: '1',
    config: storedConfig,
    createdAt: now,
    updatedAt: now,
  })

  const config = await store.getModuleConfig('warn')

  assert.deepEqual(config, storedConfig)
})

test('saveModuleConfig writes module config and audit event', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })
  const config = { threshold: TEST_THRESHOLD, mode: 'strict' }

  await store.saveModuleConfig('warn', config, 'admin')

  const [configRow] = database.tableRows(MODULE_CONFIG_TABLE)
  assert.equal(configRow.id, 'warn')
  assert.equal(configRow.moduleId, 'warn')
  assert.equal(configRow.schemaVersion, '1')
  assert.deepEqual(configRow.config, config)

  const [auditRow] = database.tableRows(AUDIT_EVENT_TABLE)
  assert.equal(typeof auditRow.id, 'string')
  assert.notEqual(auditRow.id, '')
  assert.equal(auditRow.actor, 'admin')
  assert.equal(auditRow.moduleId, 'warn')
  assert.equal(auditRow.action, 'module.config.save')
  assert.equal(auditRow.summary, '保存模块配置：warn')
  assert.deepEqual(auditRow.payload, { config })
})

test('saveModuleConfig preserves schemaVersion when updating existing row', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })
  const createdAt = new Date('2026-04-01T00:00:00.000Z')

  database.tableRows(MODULE_CONFIG_TABLE).push({
    id: 'warn',
    moduleId: 'warn',
    schemaVersion: EXISTING_SCHEMA_VERSION,
    config: { threshold: 1 },
    createdAt,
    updatedAt: createdAt,
  })

  await store.saveModuleConfig('warn', { threshold: TEST_THRESHOLD }, 'admin')

  const [configRow] = database.tableRows(MODULE_CONFIG_TABLE)
  assert.equal(configRow.schemaVersion, EXISTING_SCHEMA_VERSION)
  assert.equal(configRow.createdAt, createdAt)
  assert.deepEqual(configRow.config, { threshold: TEST_THRESHOLD })
})

test('setModuleEnabled writes disabled state', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })

  await store.setModuleEnabled('warn', false, 'admin')

  const [stateRow] = database.tableRows(MODULE_STATE_TABLE)
  assert.equal(stateRow.id, 'warn')
  assert.equal(stateRow.moduleId, 'warn')
  assert.equal(stateRow.enabled, false)
  assert.equal(stateRow.status, 'disabled')

  const [auditRow] = database.tableRows(AUDIT_EVENT_TABLE)
  assert.equal(auditRow.actor, 'admin')
  assert.equal(auditRow.moduleId, 'warn')
  assert.equal(auditRow.action, 'module.state.set')
  assert.equal(auditRow.summary, '设置模块状态：warn')
  assert.deepEqual(auditRow.payload, { enabled: false })
})

test('setModuleEnabled writes pending state when enabling module', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })

  await store.setModuleEnabled('warn', true, 'admin')

  const [stateRow] = database.tableRows(MODULE_STATE_TABLE)
  assert.equal(stateRow.id, 'warn')
  assert.equal(stateRow.moduleId, 'warn')
  assert.equal(stateRow.enabled, true)
  assert.equal(stateRow.status, 'pending')
})

test('runtime status methods write explicit module state rows', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })

  await store.markModuleLoaded({
    moduleId: 'warn',
    version: '1.2.3',
    order: 10,
    status: 'loaded',
    lastError: null,
  })

  const [stateRow] = database.tableRows(MODULE_STATE_TABLE)
  assert.equal(stateRow.enabled, true)
  assert.equal(stateRow.version, '1.2.3')
  assert.equal(stateRow.order, 10)
  assert.equal(stateRow.status, 'loaded')
  assert.equal(stateRow.lastError, null)

  await store.markModuleError({
    moduleId: 'warn',
    version: '1.2.3',
    order: 10,
    status: 'error',
    lastError: 'boom',
  })

  const [updatedRow] = database.tableRows(MODULE_STATE_TABLE)
  assert.equal(updatedRow.enabled, true)
  assert.equal(updatedRow.status, 'error')
  assert.equal(updatedRow.lastError, 'boom')

  await store.markModuleLoaded({
    moduleId: 'warn',
    version: '1.2.3',
    order: 10,
    status: 'loaded',
    lastError: null,
  })

  const [recoveredRow] = database.tableRows(MODULE_STATE_TABLE)
  assert.equal(recoveredRow.status, 'loaded')
  assert.equal(recoveredRow.lastError, null)
})

test('listAuditEvents returns newest records first with optional limit', async () => {
  const database = new FakeDatabase()
  const store = new PlatformConfigStore({ database })
  database.tableRows(AUDIT_EVENT_TABLE).push(
    auditRow('old', OLDER_AUDIT_TIME),
    auditRow('new', NEWER_AUDIT_TIME),
  )

  const records = await store.listAuditEvents(1)

  assert.deepEqual(records.map((record) => record.id), ['new'])
  assert.deepEqual(database.tableRows(AUDIT_EVENT_TABLE).map((row) => row.id), ['old', 'new'])
})

function matchesQuery(row: Row, query: Row): boolean {
  return Object.entries(query).every(([key, value]) => row[key] === value)
}

function auditRow(id: string, now: Date): Row {
  return {
    id,
    actor: 'admin',
    moduleId: 'warn',
    action: 'module.state.set',
    summary: id,
    payload: {},
    createdAt: now,
    updatedAt: now,
  }
}
