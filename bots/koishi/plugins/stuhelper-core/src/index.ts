import { Context, Logger, Schema } from 'koishi'
import type {} from '@koishijs/plugin-console'
import { ModerationStore } from '@stuhelper/koishi-moderation-core'

import {
  createCoreConfigSchema,
  type StuhelperCoreConfig,
} from '@stuhelper/koishi-shared'

import { StuhelperGroupCenterService, registerGovernanceActionAPI, registerPageAPI, registerReviewActionAPI, registerWebSocketAPI } from './core'
import { resolveBrowserEntry } from './browser-entry'
import type { BaseModule } from './core/modules'
import { WarnModule, KeywordModule, WelcomeModule, RepeatModule, DiceModule, BanmeModule, AntiRecallModule, AIModule, ConfigModule, LogModule, SubscriptionModule, HelpModule, ReportModule, GetAuthModule, AuthModule, EventModule, StatusModule,
  MemberManageModule, MessageManageModule, OrderManageModule, AntirepeatModule, crossGroupModule} from './core/modules'
import { applyLegacyFeatures } from './legacy/legacy-wrapper'
import { validateConsoleAdminPassword } from './console-auth'
import { registerReviewClaimRecovery } from './review-claim-recovery'

// 插件元信息
export const name = 'stuhelper-core'
export { usage } from './config'
export type Config = StuhelperCoreConfig
export const Config: Schema<Config> = createCoreConfigSchema()

// 声明依赖注入
export const inject = {
  required: ['database'],
  optional: ['console', 'puppeteer', 'auth']
}

// 声明服务类型扩展（注意：这里不能使用，需要在 service 文件中声明）
// declare module 'koishi' { ... } 已在 stuhelper-group-center.service.ts 中定义

const logger = new Logger('stuhelper-core')
type ModuleClass = new (...args: any[]) => BaseModule

const MODULE_CLASSES: ModuleClass[] = [
  WarnModule,
  KeywordModule,
  MemberManageModule,
  MessageManageModule,
  OrderManageModule,
  AntirepeatModule,
  WelcomeModule,
  RepeatModule,
  DiceModule,
  BanmeModule,
  AntiRecallModule,
  AIModule,
  ConfigModule,
  LogModule,
  SubscriptionModule,
  HelpModule,
  ReportModule,
  GetAuthModule,
  AuthModule,
  EventModule,
  StatusModule,
  crossGroupModule as unknown as ModuleClass,
]

/**
 * 插件入口函数
 */
export function apply(ctx: Context, config: Config) {
  ctx.plugin(StuhelperGroupCenterService)
  logger.info('StuHelper 群管中心服务已注册')
  registerConsoleEntry(ctx)
  registerConsoleAPI(ctx, config)
  registerModules(ctx)
  applyLegacyFeatures(ctx, config)
  logger.info('StuHelper 群管中心插件已加载')
}

function registerConsoleEntry(ctx: Context) {
  ctx.inject(['console'], (consoleCtx) => {
    consoleCtx.console.addEntry(resolveBrowserEntry())
    logger.info('StuHelper 群管中心控制台入口已注册')
  })
}

function registerConsoleAPI(ctx: Context, config: Config) {
  ctx.inject(['console', 'database', 'stuhelperGroupCenter', 'auth'], (apiCtx) => {
    validateConsoleAdminPassword(process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD)
    registerWebSocketAPI(apiCtx, apiCtx.stuhelperGroupCenter)
    registerPageAPI(apiCtx, {
      service: apiCtx.stuhelperGroupCenter,
      platform: config.platform,
      guard: config.guard,
    })
    registerReviewActionAPI(apiCtx)
    registerGovernanceActionAPI(apiCtx)
    logger.info('WebSocket API registered')
  })
}

function registerModules(ctx: Context) {
  ctx.inject(['database', 'stuhelperGroupCenter'], (moduleCtx) => {
    registerReviewClaimRecovery(moduleCtx, new ModerationStore(moduleCtx))
    moduleCtx.on('ready', async () => {
      const service = moduleCtx.stuhelperGroupCenter
      const moduleConfig = service.pluginConfig
      for (const ModuleType of MODULE_CLASSES) {
        service.registerModule(new ModuleType(moduleCtx, service.data, moduleConfig))
      }
      await service.initModules()
      logger.info('StuHelper 群管中心模块初始化完成')
    })
  })
}

export default {
  name,
  Config,
  apply,
}
