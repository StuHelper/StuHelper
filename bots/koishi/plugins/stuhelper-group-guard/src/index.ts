import { Context, Schema } from 'koishi'

import {
  GuardPolicyStore,
  createGroupGuardPluginConfigSchema,
  createPlatformClient,
  createPluginLogger,
  registerGuardPolicyModels,
  type StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import {
  ModerationActionService,
  ModerationStore,
  registerModerationModels,
} from '@stuhelper/koishi-moderation-core'

import { registerGuardMemberModel, type GuardMemberRecord } from './model'
import { GuardMemberStore } from './store'
import { MemberGuardService } from './member-guard'
import { MessageGuardService } from './message-guard'
import { registerGroupGuardEvents } from './events'
import { ReportService } from './report-service'
import { registerPublicCommands } from './commands'

export const name = 'stuhelper-group-guard'
export const inject = ['database']

export type Config = StuhelperGroupGuardPluginConfig

export const Config: Schema<Config> = createGroupGuardPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  registerGuardMemberModel(ctx)
  registerGuardPolicyModels(ctx)
  registerModerationModels(ctx)

  const logger = createPluginLogger(ctx, 'group-guard')
  const platform = createPlatformClient(config.platform)
  const guardStore = new GuardMemberStore(ctx)
  const policyStore = new GuardPolicyStore(ctx, config.guard)
  const moderationStore = new ModerationStore(ctx)
  const actions = new ModerationActionService(moderationStore)
  const memberGuard = new MemberGuardService({
    platform,
    guardStore,
    policyStore,
    moderationStore,
    logger,
  })
  const messageGuard = new MessageGuardService({
    store: moderationStore,
    actions,
    logger,
    config,
  })
  const reportService = new ReportService({
    store: moderationStore,
    actions,
    logger,
    config,
  })

  registerGroupGuardEvents(ctx, {
    memberGuard,
    messageGuard,
    scanIntervalSeconds: config.scheduler.scanIntervalSeconds,
  })

  ctx.on('ready', async () => {
    try {
      await messageGuard.bootstrapKeywordRules()
    } catch (error) {
      logger.error('failed to bootstrap moderation rules', error)
    }
  })

  registerPublicCommands(ctx, {
    store: moderationStore,
    reportService,
    config,
  })

  logger.info(`群管插件已加载，目标群数量：${config.guard.targetGroups.length}，扫描间隔：${config.scheduler.scanIntervalSeconds} 秒`)
}

export default {
  name,
  inject,
  Config,
  apply,
}
