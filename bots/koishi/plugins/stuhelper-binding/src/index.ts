import { Context, Schema } from 'koishi'

import {
  createBindingPluginConfigSchema,
  createPluginLogger,
  type StuhelperBindingPluginConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-binding'

export type Config = StuhelperBindingPluginConfig

export const Config: Schema<Config> = createBindingPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'binding')

  logger.info(`绑定插件已加载，命令字：${config.binding.command}`)
}
