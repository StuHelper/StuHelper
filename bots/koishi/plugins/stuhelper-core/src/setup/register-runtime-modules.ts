import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import { getRuntimeModules } from '../runtime/registry'

const logger = new Logger('stuhelper-core')

export function registerRuntimeModules(ctx: Context, config?: Config) {
  if (config?.runtimeModules?.enabled === false) {
    logger.info('StuHelper 群管中心运行时模块已关闭，仅注册 WebUI 与 Console API')
    return
  }

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
