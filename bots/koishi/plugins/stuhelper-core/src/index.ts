import { Context, Schema } from 'koishi'

import {
  createCoreConfigSchema,
  type StuhelperCoreConfig,
} from '@stuhelper/koishi-shared'

import { registerBackgroundJobs } from './setup/register-background-jobs'
import { registerConsoleApi } from './setup/register-console-api'
import { registerConsoleEntry } from './setup/register-console-entry'
import { registerCoreService } from './setup/register-core-service'
import { registerRuntimeModules } from './setup/register-runtime-modules'

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

/**
 * 插件入口函数
 */
export function apply(ctx: Context, config: Config) {
  registerCoreService(ctx)
  registerConsoleEntry(ctx)
  registerConsoleApi(ctx, config)
  registerBackgroundJobs(ctx)
  registerRuntimeModules(ctx, config)
}

export default {
  name,
  Config,
  apply,
}
