import { randomUUID } from 'node:crypto'

import {
  AUDIT_EVENT_TABLE,
  MODULE_CONFIG_TABLE,
  MODULE_STATE_TABLE,
} from './constants'
import type {
  AuditEventRecord,
  ModuleConfigRecord,
  ModuleStateRecord,
  ModuleStartupStatus,
} from './platform-models'
import type { ModuleStatusInput } from './platform-runtime'

const DEFAULT_MODULE_ORDER = 100
const DEFAULT_SCHEMA_VERSION = '1'
const INITIAL_MODULE_VERSION = '0.0.0'

type DatabaseRow = Record<string, unknown>

interface ModuleStateRowInput {
  readonly moduleId: string
  readonly enabled: boolean
  readonly status: ModuleStartupStatus
  readonly now: Date
  readonly existing: ModuleStateRecord | null
  readonly version?: string
  readonly order?: number
  readonly lastError?: string | null
}

interface ModuleConfigRowInput {
  readonly moduleId: string
  readonly config: Record<string, unknown>
  readonly now: Date
  readonly existing: ModuleConfigRecord | null
}

interface AuditEventInput {
  readonly actor: string
  readonly moduleId: string
  readonly action: string
  readonly summary: string
  readonly payload: Record<string, unknown>
}

export interface DatabaseLike {
  get(table: string, query: DatabaseRow): Promise<readonly DatabaseRow[]>
  upsert(table: string, rows: DatabaseRow[]): Promise<unknown>
  create(table: string, row: DatabaseRow): Promise<unknown>
}

export interface PlatformConfigStoreDeps {
  readonly database: DatabaseLike
}

export class PlatformConfigStore {
  private readonly database: DatabaseLike

  constructor(deps: PlatformConfigStoreDeps) {
    this.database = deps.database
  }

  async getModuleState(moduleId: string, defaultEnabled: boolean): Promise<ModuleStateRecord> {
    const record = await this.findModuleState(moduleId)
    if (record) {
      return record
    }
    return createDefaultModuleState(moduleId, defaultEnabled, new Date())
  }

  async getModuleConfig(moduleId: string): Promise<Record<string, unknown> | null> {
    const record = await this.findModuleConfig(moduleId)
    return record?.config ?? null
  }

  async saveModuleConfig(
    moduleId: string,
    config: Record<string, unknown>,
    actor: string,
  ): Promise<void> {
    const existing = await this.findModuleConfig(moduleId)
    const now = new Date()
    const row = createModuleConfigRow({ moduleId, config, now, existing })

    await this.database.upsert(MODULE_CONFIG_TABLE, [row])
    await this.appendAudit({
      actor,
      moduleId,
      action: 'module.config.save',
      summary: `保存模块配置：${moduleId}`,
      payload: { config },
    })
  }

  async setModuleEnabled(moduleId: string, enabled: boolean, actor: string): Promise<void> {
    const existing = await this.findModuleState(moduleId)
    const now = new Date()
    const status: ModuleStartupStatus = enabled ? 'pending' : 'disabled'
    const row = createModuleStateRow({ moduleId, enabled, status, now, existing })

    await this.database.upsert(MODULE_STATE_TABLE, [row])
    await this.appendAudit({
      actor,
      moduleId,
      action: 'module.state.set',
      summary: `设置模块状态：${moduleId}`,
      payload: { enabled },
    })
  }

  async markModuleLoaded(input: ModuleStatusInput): Promise<void> {
    await this.saveModuleStatus(input)
  }

  async markModuleError(input: ModuleStatusInput): Promise<void> {
    await this.saveModuleStatus(input)
  }

  async listAuditEvents(limit?: number): Promise<readonly AuditEventRecord[]> {
    const records = await this.database.get(AUDIT_EVENT_TABLE, {})
    const sorted = [...records as unknown as AuditEventRecord[]].sort(compareAuditEvents)
    return limit === undefined ? sorted : sorted.slice(0, limit)
  }

  async appendAudit(input: AuditEventInput): Promise<void> {
    const now = new Date()
    const row = {
      id: randomUUID(),
      actor: input.actor,
      moduleId: input.moduleId,
      action: input.action,
      summary: input.summary,
      payload: input.payload,
      createdAt: now,
      updatedAt: now,
    } satisfies AuditEventRecord

    await this.database.create(AUDIT_EVENT_TABLE, row)
  }

  private async findModuleState(moduleId: string): Promise<ModuleStateRecord | null> {
    const records = await this.database.get(MODULE_STATE_TABLE, { id: moduleId })
    return (records[0] as unknown as ModuleStateRecord | undefined) ?? null
  }

  private async findModuleConfig(moduleId: string): Promise<ModuleConfigRecord | null> {
    const records = await this.database.get(MODULE_CONFIG_TABLE, { id: moduleId })
    return (records[0] as unknown as ModuleConfigRecord | undefined) ?? null
  }

  private async saveModuleStatus(input: ModuleStatusInput): Promise<void> {
    const existing = await this.findModuleState(input.moduleId)
    const now = new Date()
    const row = createModuleStateRow({
      moduleId: input.moduleId,
      enabled: true,
      status: input.status,
      now,
      existing,
      version: input.version,
      order: input.order,
      lastError: input.lastError,
    })

    await this.database.upsert(MODULE_STATE_TABLE, [row])
  }
}

function createDefaultModuleState(
  moduleId: string,
  enabled: boolean,
  now: Date,
): ModuleStateRecord {
  const status: ModuleStartupStatus = enabled ? 'pending' : 'disabled'
  return {
    id: moduleId,
    moduleId,
    enabled,
    order: DEFAULT_MODULE_ORDER,
    version: INITIAL_MODULE_VERSION,
    status,
    lastError: null,
    createdAt: now,
    updatedAt: now,
  }
}

function createModuleStateRow(input: ModuleStateRowInput) {
  return {
    id: input.moduleId,
    moduleId: input.moduleId,
    enabled: input.enabled,
    order: input.order ?? input.existing?.order ?? DEFAULT_MODULE_ORDER,
    version: input.version ?? input.existing?.version ?? INITIAL_MODULE_VERSION,
    status: input.status,
    lastError: input.lastError === undefined
      ? input.existing?.lastError ?? null
      : input.lastError,
    createdAt: input.existing?.createdAt ?? input.now,
    updatedAt: input.now,
  } satisfies ModuleStateRecord
}

function createModuleConfigRow(input: ModuleConfigRowInput) {
  return {
    id: input.moduleId,
    moduleId: input.moduleId,
    schemaVersion: input.existing?.schemaVersion ?? DEFAULT_SCHEMA_VERSION,
    config: input.config,
    createdAt: input.existing?.createdAt ?? input.now,
    updatedAt: input.now,
  } satisfies ModuleConfigRecord
}

function compareAuditEvents(left: AuditEventRecord, right: AuditEventRecord): number {
  return right.createdAt.getTime() - left.createdAt.getTime()
    || right.id.localeCompare(left.id)
}
