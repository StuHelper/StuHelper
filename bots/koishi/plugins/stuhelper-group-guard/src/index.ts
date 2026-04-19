import { Context, Schema } from 'koishi'

import {
  createGroupGuardPluginConfigSchema,
  createPluginLogger,
  type StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-group-guard'

export type Config = StuhelperGroupGuardPluginConfig

export const Config: Schema<Config> = createGroupGuardPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'group-guard')
  const groupCount = config.guard.targetGroups.length

  logger.info(`群管插件已加载，目标群数量：${groupCount}，扫描间隔：${config.scheduler.scanIntervalSeconds} 秒`)
}
