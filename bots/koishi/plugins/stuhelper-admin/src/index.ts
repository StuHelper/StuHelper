import { Context, Schema } from 'koishi'

import {
  createAdminPluginConfigSchema,
  createPluginLogger,
  type StuhelperAdminPluginConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-admin'

export type Config = StuhelperAdminPluginConfig

export const Config: Schema<Config> = createAdminPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'admin')
  const state = config.admin.enableCommands ? '启用' : '停用'

  logger.info(`管理员插件已加载，命令状态：${state}`)
}
