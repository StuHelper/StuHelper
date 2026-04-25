import { Context, Schema } from 'koishi'

import {
  ModerationStore,
  registerModerationModels,
} from '@stuhelper/koishi-moderation-core'
import {
  createAdminPluginConfigSchema,
  createPluginLogger,
  registerGuardMemberModel,
  type StuhelperAdminPluginConfig,
} from '@stuhelper/koishi-shared'

import { registerAdminCommands } from './commands'

export const name = 'stuhelper-admin'
export const inject = ['database']

export type Config = StuhelperAdminPluginConfig

export const Config: Schema<Config> = createAdminPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  registerGuardMemberModel(ctx)
  registerModerationModels(ctx)

  const logger = createPluginLogger(ctx, 'admin')
  const moderationStore = new ModerationStore(ctx)

  if (!config.admin.enableCommands) {
    logger.info('管理员命令已停用')
    return
  }

  registerAdminCommands(ctx, {
    moderationStore,
  })

  logger.info('管理员命令已启用')
}

export default {
  name,
  inject,
  Config,
  apply,
}
