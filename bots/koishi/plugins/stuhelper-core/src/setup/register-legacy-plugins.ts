import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import { applyLegacyFeatures } from '../legacy/legacy-wrapper'

const logger = new Logger('stuhelper-core')

export function registerLegacyPlugins(ctx: Context, config?: Config) {
  if (!config) {
    throw new Error('stuhelper-core config is required to register legacy plugins')
  }

  applyLegacyFeatures(ctx, config)
  logger.info('StuHelper 群管中心插件已加载')
}
