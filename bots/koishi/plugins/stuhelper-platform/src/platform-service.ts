import type {
  StuhelperModuleConfig,
  StuhelperModule,
  StuhelperModuleManifest,
  StuhelperPermissionDefinition,
  StuhelperWebuiContribution,
} from './module-contract'
import type { StuhelperModuleRegistry } from './module-registry'
import type { AuditEventRecord, ModuleStartupStatus } from './platform-models'
import type { StuhelperPlatformRuntime } from './platform-runtime'

export interface PlatformServiceConfigStore {
  getModuleState(
    moduleId: string,
    defaultEnabled: boolean,
  ): Promise<{
    readonly enabled: boolean
    readonly status: ModuleStartupStatus
    readonly lastError: string | null
  }>
  getModuleConfig(moduleId: string): Promise<StuhelperModuleConfig | null>
  listAuditEvents(limit?: number): Promise<readonly AuditEventRecord[]>
  setModuleEnabled(moduleId: string, enabled: boolean, actor: string): Promise<void>
  saveModuleConfig(
    moduleId: string,
    config: StuhelperModuleConfig,
    actor: string,
  ): Promise<void>
}

export interface StuhelperPlatformServiceDeps {
  readonly registry: StuhelperModuleRegistry
  readonly store: PlatformServiceConfigStore
  readonly runtime: Pick<StuhelperPlatformRuntime, 'syncModule' | 'restartModule'>
}

export interface StuhelperModuleSnapshot {
  readonly manifest: Readonly<StuhelperModuleManifest>
  readonly enabled: boolean
  readonly status: ModuleStartupStatus
  readonly lastError: string | null
  readonly config: Readonly<StuhelperModuleConfig>
  readonly permissions: readonly Readonly<StuhelperPermissionDefinition>[]
  readonly commands: readonly StuhelperCommandSnapshot[]
  readonly events: readonly StuhelperEventSnapshot[]
  readonly webui: readonly Readonly<StuhelperWebuiContribution>[]
}

export interface StuhelperCommandSnapshot {
  readonly name: string
  readonly description: string
  readonly permission: string
}

export interface StuhelperEventSnapshot {
  readonly name: string
}

export interface StuhelperAuditEventSnapshot {
  readonly id: string
  readonly actor: string
  readonly moduleId: string
  readonly action: string
  readonly summary: string
  readonly payload: Readonly<StuhelperModuleConfig>
  readonly createdAt: string
  readonly updatedAt: string
}

export class StuhelperPlatformService {
  private readonly registry: StuhelperModuleRegistry
  private readonly store: PlatformServiceConfigStore
  private readonly runtime: Pick<StuhelperPlatformRuntime, 'syncModule' | 'restartModule'>

  constructor(deps: StuhelperPlatformServiceDeps) {
    this.registry = deps.registry
    this.store = deps.store
    this.runtime = deps.runtime
  }

  async listModules(): Promise<readonly StuhelperModuleSnapshot[]> {
    const snapshots: StuhelperModuleSnapshot[] = []

    for (const module of this.registry.list()) {
      snapshots.push(await this.createSnapshot(module))
    }

    return Object.freeze(snapshots)
  }

  async listAuditEvents(limit?: number): Promise<readonly StuhelperAuditEventSnapshot[]> {
    const records = await this.store.listAuditEvents(limit)
    return Object.freeze(records.map(serializeAuditEvent))
  }

  async setModuleEnabled(moduleId: string, enabled: boolean, actor: string): Promise<void> {
    this.requireModule(moduleId)
    await this.store.setModuleEnabled(moduleId, enabled, actor)
    await this.runtime.syncModule(moduleId)
  }

  async saveModuleConfig(moduleId: string, config: unknown, actor: string): Promise<void> {
    const module = this.requireModule(moduleId)
    const parsedConfig = module.configSchema.parse(config)
    await this.store.saveModuleConfig(moduleId, parsedConfig, actor)
    await this.runtime.restartModule(moduleId)
  }

  private async createSnapshot<TConfig extends StuhelperModuleConfig>(
    module: StuhelperModule<TConfig>,
  ): Promise<StuhelperModuleSnapshot> {
    const manifest = module.manifest
    const state = await this.store.getModuleState(manifest.id, manifest.defaultEnabled)
    const config = await this.resolveConfig(module)

    return Object.freeze({
      manifest: Object.freeze({ ...manifest }),
      enabled: state.enabled,
      status: state.status,
      lastError: state.lastError,
      config,
      permissions: freezeMetadata(module.permissions),
      commands: serializeCommands(module.commands),
      events: serializeEvents(module.events),
      webui: freezeMetadata(module.webui),
    })
  }

  private async resolveConfig(module: StuhelperModule): Promise<Readonly<StuhelperModuleConfig>> {
    const storedConfig = await this.store.getModuleConfig(module.manifest.id)
    const config = module.configSchema.parse(storedConfig ?? module.configSchema.defaults())

    return deepFreezeConfig(cloneConfig(config))
  }

  private requireModule(moduleId: string): StuhelperModule {
    const module = this.registry.get(moduleId)
    if (!module) {
      throw new Error(`unknown StuHelper module id: ${moduleId}`)
    }
    return module
  }
}

function freezeMetadata<TItem extends object>(items: readonly TItem[]): readonly Readonly<TItem>[] {
  return Object.freeze(items.map((item) => Object.freeze({ ...item })))
}

function serializeCommands(
  items: readonly { readonly name: string; readonly description: string; readonly permission: string }[],
): readonly StuhelperCommandSnapshot[] {
  return Object.freeze(items.map((item) => Object.freeze({
    name: item.name,
    description: item.description,
    permission: item.permission,
  })))
}

function serializeEvents(items: readonly { readonly name: string }[]): readonly StuhelperEventSnapshot[] {
  return Object.freeze(items.map((item) => Object.freeze({ name: item.name })))
}

function cloneConfig<TConfig extends StuhelperModuleConfig>(config: TConfig): TConfig {
  return structuredClone(config)
}

function deepFreezeConfig<TValue>(value: TValue): Readonly<TValue> {
  if (Array.isArray(value)) {
    value.forEach((item) => deepFreezeConfig(item))
    return Object.freeze(value) as Readonly<TValue>
  }

  if (isConfigObject(value)) {
    Object.values(value).forEach((item) => deepFreezeConfig(item))
    return Object.freeze(value) as Readonly<TValue>
  }

  return value as Readonly<TValue>
}

function isConfigObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function serializeAuditEvent(record: AuditEventRecord): StuhelperAuditEventSnapshot {
  return Object.freeze({
    id: record.id,
    actor: record.actor,
    moduleId: record.moduleId,
    action: record.action,
    summary: record.summary,
    payload: deepFreezeConfig(cloneConfig(record.payload)),
    createdAt: record.createdAt.toISOString(),
    updatedAt: record.updatedAt.toISOString(),
  })
}
