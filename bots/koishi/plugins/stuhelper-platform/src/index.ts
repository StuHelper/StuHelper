import { resolve } from 'node:path'

import { Context } from 'koishi'
import type {} from '@koishijs/plugin-console'

import { PlatformConfigStore } from './config-store'
import {
  registerPlatformConsoleRoutes,
  StuhelperPlatformDataService,
} from './console-routes'
import { createModuleRegistry } from './module-registry'
import { groupGuardModule } from './modules/group-guard'
import { registerPlatformModels } from './platform-models'
import { StuhelperPlatformRuntime } from './platform-runtime'
import { StuhelperPlatformService } from './platform-service'

export * from './config-store'
export * from './console-routes'
export * from './module-contract'
export * from './module-registry'
export * from './modules/group-guard'
export * from './platform-runtime'
export * from './platform-service'
export * from './platform-models'

export const name = 'stuhelper-platform'

export const inject = {
  required: ['database'],
  optional: ['console'],
}

export async function apply(ctx: Context) {
  registerPlatformModels(ctx)
  const registry = createModuleRegistry([groupGuardModule])
  const store = new PlatformConfigStore({ database: ctx.database })
  const runtime = new StuhelperPlatformRuntime({ koishi: ctx, registry, store })
  const service = new StuhelperPlatformService({ registry, store, runtime })

  for (const module of registry.list()) {
    module.prepare?.(ctx)
  }

  await runtime.start()

  ctx.inject(['console'], (consoleCtx) => {
    consoleCtx.console.addEntry({
      dev: resolve(__dirname, '../client/index.ts'),
      prod: resolve(__dirname, '../dist'),
    })
    consoleCtx.plugin(StuhelperPlatformDataService, { service })
    registerPlatformConsoleRoutes(consoleCtx, service)
  })
}

export function createPlatformService(ctx: Context): StuhelperPlatformService {
  const store = new PlatformConfigStore({ database: ctx.database })
  const registry = createModuleRegistry([groupGuardModule])
  const runtime = new StuhelperPlatformRuntime({ koishi: ctx, registry, store })
  return new StuhelperPlatformService({
    registry,
    store,
    runtime,
  })
}

export default {
  name,
  inject,
  apply,
}
