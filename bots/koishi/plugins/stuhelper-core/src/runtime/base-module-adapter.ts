import type { Context } from 'koishi'

import type { DataManager } from '../core/data'
import type { BaseModule } from '../core/modules'
import type { Config } from '../types'
import type { ModuleDeps, RuntimeModule } from './types'

export type BaseModuleClass = new (
  ctx: Context,
  data: DataManager,
  config: Config
) => BaseModule

export interface BaseModuleDefinition {
  readonly id: string
  readonly order: number
  readonly ModuleType: BaseModuleClass
}

export function adaptBaseModule(definition: BaseModuleDefinition): RuntimeModule<BaseModule> {
  return {
    id: definition.id,
    order: definition.order,
    create(ctx, deps) {
      return new definition.ModuleType(ctx, deps.data, deps.config)
    },
  }
}
