import type { Context } from 'koishi'

import type { StuhelperModule, StuhelperModuleConfig } from './module-contract'
import type { StuhelperModuleRegistry } from './module-registry'
import type { ModuleStartupStatus } from './platform-models'

const SYSTEM_ACTOR = 'system'

interface ModuleScope {
  dispose(): boolean
}

export interface PlatformRuntimeStore {
  getModuleState(
    moduleId: string,
    defaultEnabled: boolean,
  ): Promise<{ readonly enabled: boolean }>
  getModuleConfig(moduleId: string): Promise<StuhelperModuleConfig | null>
  markModuleLoaded(input: ModuleStatusInput): Promise<void>
  markModuleError(input: ModuleStatusInput): Promise<void>
  appendAudit(input: ModuleAuditInput): Promise<void>
}

export interface ModuleStatusInput {
  readonly moduleId: string
  readonly version: string
  readonly order: number
  readonly status: ModuleStartupStatus
  readonly lastError: string | null
}

export interface ModuleAuditInput {
  readonly actor: string
  readonly moduleId: string
  readonly action: string
  readonly summary: string
  readonly payload: Record<string, unknown>
}

export interface StuhelperPlatformRuntimeDeps {
  readonly koishi: Context
  readonly registry: StuhelperModuleRegistry
  readonly store: PlatformRuntimeStore
}

export class StuhelperPlatformRuntime {
  private readonly koishi: Context
  private readonly registry: StuhelperModuleRegistry
  private readonly store: PlatformRuntimeStore
  private readonly activeModules = new Map<string, ModuleScope>()

  constructor(deps: StuhelperPlatformRuntimeDeps) {
    this.koishi = deps.koishi
    this.registry = deps.registry
    this.store = deps.store
  }

  async start(): Promise<void> {
    for (const module of this.registry.list()) {
      await this.syncModule(module.manifest.id)
    }
  }

  async syncModule(moduleId: string): Promise<void> {
    const module = this.requireModule(moduleId)
    const state = await this.store.getModuleState(
      module.manifest.id,
      module.manifest.defaultEnabled,
    )
    if (!state.enabled) {
      this.stopModule(module.manifest.id)
      return
    }
    if (this.activeModules.has(module.manifest.id)) return
    await this.startModule(module)
  }

  async restartModule(moduleId: string): Promise<void> {
    this.stopModule(moduleId)
    await this.syncModule(moduleId)
  }

  private async startModule(module: StuhelperModule): Promise<void> {
    let moduleContext: Context | null = null
    const fork = this.koishi.plugin((ctx) => {
      moduleContext = ctx
    })

    try {
      if (!moduleContext) {
        throw new Error(`failed to create StuHelper module scope: ${module.manifest.id}`)
      }
      await module.setup({
        koishi: moduleContext,
        config: await this.resolveConfig(module),
        audit: (event) => this.store.appendAudit({ actor: SYSTEM_ACTOR, ...event }),
      })
      this.activeModules.set(module.manifest.id, fork)
      await this.store.markModuleLoaded(createStatusInput(module, 'loaded', null))
    } catch (error) {
      fork.dispose()
      const message = error instanceof Error ? error.message : String(error)
      await this.store.markModuleError(createStatusInput(module, 'error', message))
      throw error
    }
  }

  private async resolveConfig(module: StuhelperModule): Promise<StuhelperModuleConfig> {
    const storedConfig = await this.store.getModuleConfig(module.manifest.id)
    return module.configSchema.parse(storedConfig ?? module.configSchema.defaults())
  }

  private stopModule(moduleId: string): void {
    const fork = this.activeModules.get(moduleId)
    if (!fork) return
    fork.dispose()
    this.activeModules.delete(moduleId)
  }

  private requireModule(moduleId: string): StuhelperModule {
    const module = this.registry.get(moduleId)
    if (!module) {
      throw new Error(`unknown StuHelper module id: ${moduleId}`)
    }
    return module
  }
}

function createStatusInput(
  module: StuhelperModule,
  status: ModuleStartupStatus,
  lastError: string | null,
): ModuleStatusInput {
  return {
    moduleId: module.manifest.id,
    version: module.manifest.version,
    order: module.manifest.order,
    status,
    lastError,
  }
}
