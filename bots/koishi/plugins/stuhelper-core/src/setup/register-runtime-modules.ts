import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import type { BaseModule } from '../core/modules'
import {
  AIModule,
  AntiRecallModule,
  AntirepeatModule,
  AuthModule,
  BanmeModule,
  ConfigModule,
  DiceModule,
  EventModule,
  GetAuthModule,
  HelpModule,
  KeywordModule,
  LogModule,
  MemberManageModule,
  MessageManageModule,
  OrderManageModule,
  RepeatModule,
  ReportModule,
  StatusModule,
  SubscriptionModule,
  WarnModule,
  WelcomeModule,
  crossGroupModule,
} from '../core/modules'

const logger = new Logger('stuhelper-core')
type ModuleClass = new (...args: any[]) => BaseModule

export const MODULE_CLASSES: ModuleClass[] = [
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

export function registerRuntimeModules(ctx: Context, _config?: Config) {
  ctx.inject(['database', 'stuhelperGroupCenter'], (moduleCtx) => {
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
