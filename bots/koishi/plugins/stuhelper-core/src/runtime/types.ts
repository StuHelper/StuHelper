import type { Command, Context } from 'koishi'

import type { DataManager } from '../core/data'
import type { StuhelperGroupCenterService } from '../core/services'
import type { Config } from '../types'

export interface RuntimeModuleMeta {
  name: string
  description: string
  version?: string
  author?: string
}

export interface RuntimeCommandDef {
  name: string
  desc: string
  args?: string
  permNode?: string
  permDesc?: string
  skipAuth?: boolean
  usage?: string
  examples?: string[]
}

export type RuntimeModuleState = 'unloaded' | 'loading' | 'loaded' | 'error'

export interface RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta
  readonly state: RuntimeModuleState
  readonly error: Error | null
  init(): Promise<void> | void
  dispose(): Promise<void> | void
}

export interface ModuleDeps {
  readonly service: StuhelperGroupCenterService
  readonly data: DataManager
  readonly config: Config
}

export interface RuntimeModule<TInstance extends RuntimeModuleInstance = RuntimeModuleInstance> {
  readonly id: string
  readonly order?: number
  create(ctx: Context, deps: ModuleDeps): TInstance
}

export type RuntimeCommand = Command
