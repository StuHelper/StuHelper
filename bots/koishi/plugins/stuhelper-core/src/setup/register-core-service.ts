import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import { StuhelperGroupCenterService } from '../core'

const logger = new Logger('stuhelper-core')

export function registerCoreService(ctx: Context, _config?: Config) {
  ctx.plugin(StuhelperGroupCenterService)
  logger.info('StuHelper 群管中心服务已注册')
}
