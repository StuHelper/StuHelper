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
import { registerAdmissionAdminCommands } from './admission-admin-commands'
import { AdmissionReminderDeduper } from './admission-reminder-deduper'
import { registerAdmissionActionStreams } from './admission-action-stream'

export const name = 'stuhelper-group-guard'
export const inject = ['database']

export type Config = StuhelperGroupGuardPluginConfig

export const Config: Schema<Config> = createGroupGuardPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  registerGroupGuardRuntimeModels(ctx)
  startGroupGuardRuntime(ctx, config)
}

export function registerGroupGuardRuntimeModels(ctx: Context) {
  registerGuardMemberModel(ctx)
  registerGuardPolicyModels(ctx)
  registerModerationModels(ctx)
}

export function startGroupGuardRuntime(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'group-guard')
  const platform = createPlatformClient(config.platform)
  const guardStore = new GuardMemberStore(ctx)
  const policyStore = new GuardPolicyStore(ctx, config.guard)
  const moderationStore = new ModerationStore(ctx)
  const actions = new ModerationActionService(moderationStore)
  const admissionReminderDeduper = new AdmissionReminderDeduper()
  const memberGuard = new MemberGuardService({
    platform,
    guardStore,
    policyStore,
    moderationStore,
    logger,
    freshmanForwardEnabled: config.freshmanForward?.enabled !== false,
    reminderDeduper: admissionReminderDeduper,
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
    messageGuard: config.moderation.enabled === false ? undefined : messageGuard,
    logger,
    scanIntervalSeconds: config.scheduler.scanIntervalSeconds,
    fallbackScanEnabled: config.scheduler.fallbackScanEnabled !== false,
  })

  registerAdmissionActionStreams(ctx, {
    platform,
    memberGuard,
    logger,
    config: config.actionStream,
  })

  if (config.moderation.enabled !== false) {
    ctx.on('ready', async () => {
      try {
        await messageGuard.bootstrapKeywordRules()
      } catch (error) {
        logger.error('failed to bootstrap moderation rules', error)
      }
    })
  }

  if (config.commands?.enabled !== false) {
    registerPublicCommands(ctx, {
      store: moderationStore,
      reportService,
      config,
    })
  }

  if (config.admissionCommands?.enabled !== false) {
    registerAdmissionAdminCommands(ctx, {
      platform,
      guardStore,
      policyStore,
      config,
      reminderDeduper: admissionReminderDeduper,
    })
  }

  logger.info(`群管插件已加载，目标群数量：${config.guard.targetGroups.length}，action stream：${config.actionStream?.enabled !== false ? 'enabled' : 'disabled'}，兜底扫描间隔：${config.scheduler.scanIntervalSeconds} 秒`)
}

export default {
  name,
  inject,
  Config,
  apply,
}
