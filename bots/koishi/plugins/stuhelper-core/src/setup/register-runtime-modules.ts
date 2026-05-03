import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import { getRuntimeModules } from '../runtime/registry'

const logger = new Logger('stuhelper-core')

export function registerRuntimeModules(ctx: Context, config?: Config) {
  ctx.inject(['database', 'stuhelperGroupCenter'], (moduleCtx) => {
    moduleCtx.on('ready', async () => {
      const service = moduleCtx.stuhelperGroupCenter
      const deps = {
        service,
        data: service.data,
        config: service.pluginConfig,
        coreConfig: config,
      }
      for (const runtimeModule of getRuntimeModules()) {
        service.registerModule(runtimeModule.create(moduleCtx, deps))
      }
      await service.initModules()
      logger.info('StuHelper 群管中心模块初始化完成')
    })
  })
}
