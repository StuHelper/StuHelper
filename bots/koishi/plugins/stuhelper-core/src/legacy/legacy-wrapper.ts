import { Context, Schema } from 'koishi'

import {
  createCoreConfigSchema,
  createPluginLogger,
  type StuhelperCoreConfig,
} from '@stuhelper/koishi-shared'
import * as admin from 'koishi-plugin-stuhelper-admin'
import * as binding from 'koishi-plugin-stuhelper-binding'
import * as groupGuard from 'koishi-plugin-stuhelper-group-guard'

export const name = 'stuhelper-core'

export type Config = StuhelperCoreConfig

export const Config: Schema<Config> = createCoreConfigSchema()

export function applyLegacyFeatures(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'core')

  ctx.plugin(binding, {
    platform: config.platform,
    binding: config.binding,
  })
  ctx.plugin(groupGuard, {
    platform: config.platform,
    guard: config.guard,
    scheduler: config.scheduler,
    moderation: config.moderation,
    fun: config.fun,
    ai: config.ai,
  })
  ctx.plugin(admin, {
    platform: config.platform,
    admin: config.admin,
    moderation: config.moderation,
    fun: config.fun,
  })

  logger.info(`StuHelper 旧能力已并入新群管中心，平台地址：${config.platform.baseUrl}`)
}

export default {
  name,
  Config,
  apply: applyLegacyFeatures,
}
