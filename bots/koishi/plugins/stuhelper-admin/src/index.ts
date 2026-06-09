import { Context, Schema } from 'koishi'

import {
  ModerationStore,
  registerModerationModels,
} from '@stuhelper/koishi-moderation-core'
import {
  AdmissionRuntimeSettingsStore,
  AdminRuntimeSettingsStore,
  DEFAULT_ADMISSION_RUNTIME_SETTINGS,
  DEFAULT_ADMIN_RUNTIME_SETTINGS,
  createAdminPluginConfigSchema,
  createPlatformClient,
  createPluginLogger,
  GuardMemberAdminStore,
  registerAdminRuntimeSettingsModel,
  registerAdmissionRuntimeSettingsModel,
  registerGuardMemberModel,
  syncAdminCommandDescriptions,
  type StuhelperAdminPluginConfig,
} from '@stuhelper/koishi-shared'

import { registerAdminCommands } from './commands'

export const name = 'stuhelper-admin'
export const inject = ['database']

export type Config = StuhelperAdminPluginConfig

export const Config: Schema<Config> = createAdminPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  registerGuardMemberModel(ctx)
  registerAdmissionRuntimeSettingsModel(ctx)
  registerAdminRuntimeSettingsModel(ctx)
  registerModerationModels(ctx)

  const logger = createPluginLogger(ctx, 'admin')
  const guardMembers = new GuardMemberAdminStore(ctx)
  const moderationStore = new ModerationStore(ctx)
  const runtimeSettings = new AdmissionRuntimeSettingsStore(ctx, DEFAULT_ADMISSION_RUNTIME_SETTINGS)
  const adminSettings = new AdminRuntimeSettingsStore(ctx, DEFAULT_ADMIN_RUNTIME_SETTINGS)
  const platform = createPlatformClient(config.platform)

  registerAdminCommands(ctx, {
    guardMembers,
    moderationStore,
    platform,
    runtimeSettings,
    adminSettings,
  })

  logger.info('管理员命令已注册，执行开关和提示文案由 StuHelper WebUI runtime settings 控制')
  ctx.on('ready', () => {
    void syncRuntimeCommandDescriptions()
  })

  async function syncRuntimeCommandDescriptions() {
    try {
      syncAdminCommandDescriptions(ctx, await adminSettings.getMessages())
    } catch (error) {
      logger.warn('failed to sync admin command descriptions from runtime settings', error)
    }
  }
}

export default {
  name,
  inject,
  Config,
  apply,
}
