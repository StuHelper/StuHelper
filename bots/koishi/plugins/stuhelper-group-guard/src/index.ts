import { Context, Schema } from 'koishi'

import {
  AdmissionRuntimeSettingsStore,
  DEFAULT_ADMISSION_RUNTIME_SETTINGS,
  DEFAULT_GROUP_GUARD_AI_SETTINGS,
  DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS,
  GroupGuardAISettingsStore,
  GroupGuardBehaviorSettingsStore,
  GroupGuardMessageSettingsStore,
  GuardMemberStore,
  GuardPolicyStore,
  createGroupGuardPluginConfigSchema,
  createPlatformClient,
  createPluginLogger,
  registerAdmissionRuntimeSettingsModel,
  registerGroupGuardAISettingsModel,
  registerGroupGuardBehaviorSettingsModel,
  registerGroupGuardMessageSettingsModel,
  registerGuardMemberModel,
  registerGuardPolicyModels,
  syncGroupGuardCommandDescriptions,
  type StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'
import {
  ModerationActionService,
  ModerationStore,
  registerModerationModels,
} from '@stuhelper/koishi-moderation-core'

import { MemberGuardService } from './member-guard'
import { MessageGuardService } from './message-guard'
import { registerGroupGuardEvents } from './events'
import { ReportService } from './report-service'
import { registerPublicCommands } from './commands'
import { registerAdmissionAdminCommands } from './admission-admin-commands'
import { AdmissionReminderDeduper } from './admission-reminder-deduper'
import { AdmissionSubjectCoordinator } from './admission-subject-coordinator'
import { registerAdmissionActionStreams } from './admission-action-stream'
import {
  registerAdmissionConsoleAPI,
  type AdmissionConsoleGuildScope,
} from './admission-console-api'
import { syncGuardPolicyFromAdmissionTargets } from './guard-policy-bootstrap'
import { createGroupGuardAISettingsProvider } from './group-guard-ai-provider'
import { createGroupGuardMessageProvider } from './group-guard-message-provider'
import { registerAlertmanagerWebhook } from './alertmanager-webhook'

export const name = 'stuhelper-group-guard'
export const inject = {
  required: ['database'],
  optional: ['console', 'server', 'stuhelperGroupCenter'],
}

export type Config = StuhelperGroupGuardPluginConfig

export const Config: Schema<Config> = createGroupGuardPluginConfigSchema()

export function apply(ctx: Context, config: Config | null | undefined) {
  if (!config) {
    throw new Error('stuhelper_group_guard_configuration_invalid')
  }
  registerGroupGuardRuntimeModels(ctx)
  startGroupGuardRuntime(ctx, config)
}

export function registerGroupGuardRuntimeModels(ctx: Context) {
  registerGuardMemberModel(ctx)
  registerGuardPolicyModels(ctx)
  registerAdmissionRuntimeSettingsModel(ctx)
  registerGroupGuardAISettingsModel(ctx)
  registerGroupGuardBehaviorSettingsModel(ctx)
  registerGroupGuardMessageSettingsModel(ctx)
  registerModerationModels(ctx)
}

export function startGroupGuardRuntime(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'group-guard')
  const platform = createPlatformClient(config.platform)
  const guardStore = new GuardMemberStore(ctx)
  const policyStore = new GuardPolicyStore(ctx)
  const runtimeSettings = new AdmissionRuntimeSettingsStore(ctx, DEFAULT_ADMISSION_RUNTIME_SETTINGS)
  const aiSettings = new GroupGuardAISettingsStore(ctx, DEFAULT_GROUP_GUARD_AI_SETTINGS)
  const behaviorSettings = new GroupGuardBehaviorSettingsStore(ctx)
  const messageSettings = new GroupGuardMessageSettingsStore(ctx, DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS)
  const aiSettingsProvider = createGroupGuardAISettingsProvider(aiSettings)
  const messageProvider = createGroupGuardMessageProvider(messageSettings)
  const moderationStore = new ModerationStore(ctx)
  const actions = new ModerationActionService(moderationStore, messageProvider)
  const admissionReminderDeduper = new AdmissionReminderDeduper()
  const admissionSubjectCoordinator = new AdmissionSubjectCoordinator()
  const memberGuard = new MemberGuardService({
    platform,
    guardStore,
    policyStore,
    moderationStore,
    logger,
    isTimeCodeReminderEnabled: () => runtimeSettings.isTimeCodeReminderEnabled(),
    admissionSubjectCoordinator,
    reminderDeduper: admissionReminderDeduper,
    admissionReminderDelivery: () => runtimeSettings.getAdmissionReminderDeliveryConfig(),
    messageProvider,
  })
  const messageGuard = new MessageGuardService({
    store: moderationStore,
    actions,
    logger,
    behaviorSettings,
    messageProvider,
  })
  const reportService = new ReportService({
    store: moderationStore,
    actions,
    logger,
    aiSettings: aiSettingsProvider,
    behaviorSettings,
    messageProvider,
  })

  if (config.alerting?.enabled) {
    registerAlertmanagerWebhook(ctx, config.alerting, {
      platform,
      getBots: () => ctx.bots,
    }, logger)
  }

  registerGroupGuardEvents(ctx, {
    memberGuard,
    messageGuard,
    runtimeSettings,
    logger,
    scanIntervalSeconds: config.scheduler.scanIntervalSeconds,
  })

  const actionStreams = registerAdmissionActionStreams(ctx, {
    platform,
    memberGuard,
    logger,
    config: config.actionStream,
    isEnabled: () => runtimeSettings.isActionStreamEnabled(),
  })

  registerPublicCommands(ctx, {
    store: moderationStore,
    reportService,
    runtimeSettings,
    behaviorSettings,
    messageProvider,
  })

  registerAdmissionAdminCommands(ctx, {
    platform,
    guardStore,
    policyStore,
    moderationStore,
    config,
    runtimeSettings,
    messageProvider,
    admissionSubjectCoordinator,
    reminderDeduper: admissionReminderDeduper,
  })

  ctx.inject(['console', 'stuhelperGroupCenter'], (consoleCtx) => {
    registerAdmissionConsoleAPI(consoleCtx, {
      config,
      platform,
      runtimeSettings,
      behaviorSettings,
      messageProvider,
      guardStore,
      policyStore,
      moderationStore,
      admissionSubjectCoordinator,
      onRuntimeSettingsChanged: () => actionStreams.refresh(),
      resolveConsoleScope: (client) => {
        const service = (consoleCtx as Context & {
          stuhelperGroupCenter: {
            resolveConsoleGuildScope(client: unknown): Promise<AdmissionConsoleGuildScope>
          }
        }).stuhelperGroupCenter
        return service.resolveConsoleGuildScope(client)
      },
    })
  })

  ctx.on('ready', async () => {
    await syncBackendAdmissionPolicyTargets()
  })

  ctx.setInterval(() => {
    void syncBackendAdmissionPolicyTargets()
  }, Math.max(60, config.scheduler.scanIntervalSeconds) * 1000)

  logger.info(`群管插件已加载，目标群由后端 admission policy 同步为 Koishi 执行态缓存，action stream 和兜底扫描开关由 WebUI 控制，兜底扫描间隔：${config.scheduler.scanIntervalSeconds} 秒`)
  ctx.on('ready', () => {
    void syncRuntimeCommandDescriptions()
  })

  async function syncBackendAdmissionPolicyTargets() {
    try {
      const targets = await platform.listAdmissionPolicyTargets()
      await syncGuardPolicyFromAdmissionTargets(policyStore, targets, logger)
    } catch (error) {
      logger.warn('failed to sync guard policy from backend admission policies', error)
    }
  }

  async function syncRuntimeCommandDescriptions() {
    try {
      syncGroupGuardCommandDescriptions(ctx, await messageSettings.getMessages())
    } catch (error) {
      logger.warn('failed to sync group guard command descriptions from runtime settings', error)
    }
  }
}

export default {
  name,
  inject,
  Config,
  apply,
}
