import { Context, Logger, Schema } from 'koishi'
import type {} from '@koishijs/plugin-console'

import {
  createCoreConfigSchema,
  type StuhelperCoreConfig,
} from '@stuhelper/koishi-shared'

import { StuhelperGroupCenterService, registerWebSocketAPI } from './core'
import { resolveBrowserEntry } from './browser-entry'
import { WarnModule, KeywordModule, WelcomeModule, RepeatModule, DiceModule, BanmeModule, AntiRecallModule, AIModule, ConfigModule, LogModule, SubscriptionModule, HelpModule, ReportModule, GetAuthModule, AuthModule, EventModule, StatusModule,
  MemberManageModule, MessageManageModule, OrderManageModule, AntirepeatModule, crossGroupModule} from './core/modules'
import { applyLegacyFeatures } from './legacy/legacy-wrapper'

// 插件元信息
export const name = 'stuhelper-core'
export { usage } from './config'
export type Config = StuhelperCoreConfig
export const Config: Schema<Config> = createCoreConfigSchema()

// 声明依赖注入
export const inject = {
  required: ['database'],
  optional: ['console', 'puppeteer']
}

// 声明服务类型扩展（注意：这里不能使用，需要在 service 文件中声明）
// declare module 'koishi' { ... } 已在 stuhelper-group-center.service.ts 中定义

const logger = new Logger('stuhelper-core')

/**
 * 插件入口函数
 */
export function apply(ctx: Context, config: Config) {
  // ===== 注册核心服务 =====
  ctx.plugin(StuhelperGroupCenterService)
  logger.info('StuHelper 群管中心服务已注册')

  // ===== 注册控制台页面（使用官方推荐的 inject 模式） =====
  ctx.inject(['console'], (ctx) => {
    ctx.console.addEntry(resolveBrowserEntry())
    logger.info('StuHelper 群管中心控制台入口已注册')
  })

  // ===== 注册模块和 API（确保 stuhelperGroupCenter 服务已注册后） =====
  ctx.inject(['stuhelperGroupCenter'], (ctx) => {
    // 注册 WebSocket API（如果控制台可用）
    ctx.inject(['console'], (ctx) => {
      registerWebSocketAPI(ctx, ctx.stuhelperGroupCenter)
      logger.info('WebSocket API registered')
    })

    // 在 ready 事件中初始化模块
    ctx.on('ready', async () => {
      // 注册并初始化新架构模块
      // 获取配置
      const config = ctx.stuhelperGroupCenter.pluginConfig

      const warnModule = new WarnModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const keywordModule = new KeywordModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const memberManageModule = new MemberManageModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const messageManageModule = new MessageManageModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const orderManageModule = new OrderManageModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const antiRepeatModule = new AntirepeatModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const welcomeModule = new WelcomeModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const repeatModule = new RepeatModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const diceModule = new DiceModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const banmeModule = new BanmeModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const antiRecallModule = new AntiRecallModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const aiModule = new AIModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const configModule = new ConfigModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const logModule = new LogModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const subscriptionModule = new SubscriptionModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const helpModule = new HelpModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const reportModule = new ReportModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const getAuthModule = new GetAuthModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const authModule = new AuthModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const eventModule = new EventModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const statusModule = new StatusModule(ctx, ctx.stuhelperGroupCenter.data, config)
      const crossGroupManageModule = new crossGroupModule(ctx, ctx.stuhelperGroupCenter.data, config)
      ctx.stuhelperGroupCenter.registerModule(warnModule)
      ctx.stuhelperGroupCenter.registerModule(keywordModule)
      ctx.stuhelperGroupCenter.registerModule(memberManageModule)
      ctx.stuhelperGroupCenter.registerModule(messageManageModule)
      ctx.stuhelperGroupCenter.registerModule(orderManageModule)
      ctx.stuhelperGroupCenter.registerModule(antiRepeatModule)
      ctx.stuhelperGroupCenter.registerModule(welcomeModule)
      ctx.stuhelperGroupCenter.registerModule(repeatModule)
      ctx.stuhelperGroupCenter.registerModule(diceModule)
      ctx.stuhelperGroupCenter.registerModule(banmeModule)
      ctx.stuhelperGroupCenter.registerModule(antiRecallModule)
      ctx.stuhelperGroupCenter.registerModule(aiModule)
      ctx.stuhelperGroupCenter.registerModule(configModule)
      ctx.stuhelperGroupCenter.registerModule(logModule)
      ctx.stuhelperGroupCenter.registerModule(subscriptionModule)
      ctx.stuhelperGroupCenter.registerModule(helpModule)
      ctx.stuhelperGroupCenter.registerModule(reportModule as any)
      ctx.stuhelperGroupCenter.registerModule(getAuthModule)
      ctx.stuhelperGroupCenter.registerModule(authModule)
      ctx.stuhelperGroupCenter.registerModule(eventModule)
      ctx.stuhelperGroupCenter.registerModule(statusModule)
      ctx.stuhelperGroupCenter.registerModule(crossGroupManageModule)

      // 初始化所有模块
      await ctx.stuhelperGroupCenter.initModules()
      logger.info('StuHelper 群管中心模块初始化完成')
    })
  })

  applyLegacyFeatures(ctx, config)

  logger.info('StuHelper 群管中心插件已加载')
}

export default {
  name,
  Config,
  apply,
}
