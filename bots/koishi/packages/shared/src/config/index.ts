import { Schema } from 'koishi'

import type {
  StuhelperAdminConfig,
  StuhelperAdminPluginConfig,
  StuhelperBindingConfig,
  StuhelperBindingPluginConfig,
  StuhelperCoreConfig,
  StuhelperGuardConfig,
  StuhelperGroupGuardPluginConfig,
  StuhelperPlatformConfig,
  StuhelperSchedulerConfig,
} from '../types'

const DEFAULT_BINDING_COMMAND = '绑定'
const DEFAULT_CODE_TTL_MINUTES = 10
const DEFAULT_MUTE_DURATION_SECONDS = 600
const DEFAULT_KICK_AFTER_MINUTES = 30
const DEFAULT_SCAN_INTERVAL_SECONDS = 60
const DEFAULT_REMINDER_TEMPLATE = '请先完成 StuHelper 注册、QQ 绑定与学生认证。'
const DEFAULT_PLATFORM_BASE_URL = 'http://127.0.0.1:8080'

export function createPlatformConfigSchema(): Schema<StuhelperPlatformConfig> {
  return Schema.object({
    baseUrl: Schema.string().default(DEFAULT_PLATFORM_BASE_URL).description('StuHelper 平台 API 根地址。'),
    serviceToken: Schema.string().default('').description('机器人访问 StuHelper 平台的服务令牌。'),
  })
}

export function createBindingConfigSchema(): Schema<StuhelperBindingConfig> {
  return Schema.object({
    command: Schema.string().default(DEFAULT_BINDING_COMMAND).description('用户私聊机器人的绑定命令字。'),
    codeTtlMinutes: Schema.number().min(1).default(DEFAULT_CODE_TTL_MINUTES).description('绑定码有效期，单位为分钟。'),
  })
}

export function createGuardConfigSchema(): Schema<StuhelperGuardConfig> {
  return Schema.object({
    targetGroups: Schema.array(Schema.string()).default([]).description('启用群管策略的 QQ 群号列表。'),
    muteDurationSeconds: Schema.number().min(1).default(DEFAULT_MUTE_DURATION_SECONDS).description('未认证成员首次禁言时长，单位为秒。'),
    kickAfterMinutes: Schema.number().min(1).default(DEFAULT_KICK_AFTER_MINUTES).description('未完成认证的宽限期，单位为分钟。'),
    reminderTemplate: Schema.string().default(DEFAULT_REMINDER_TEMPLATE).description('未认证成员进群后发送的提醒文案。'),
    exemptUsers: Schema.array(Schema.string()).default([]).description('跳过群管策略的 QQ 号白名单。'),
  })
}

export function createAdminConfigSchema(): Schema<StuhelperAdminConfig> {
  return Schema.object({
    enableCommands: Schema.boolean().default(true).description('是否启用 StuHelper 群管管理员命令。'),
  })
}

export function createSchedulerConfigSchema(): Schema<StuhelperSchedulerConfig> {
  return Schema.object({
    scanIntervalSeconds: Schema.number().min(1).default(DEFAULT_SCAN_INTERVAL_SECONDS).description('后台扫描待认证成员的间隔，单位为秒。'),
  })
}

export function createCoreConfigSchema(): Schema<StuhelperCoreConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    binding: createBindingConfigSchema(),
    guard: createGuardConfigSchema(),
    admin: createAdminConfigSchema(),
    scheduler: createSchedulerConfigSchema(),
  })
}

export function createBindingPluginConfigSchema(): Schema<StuhelperBindingPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    binding: createBindingConfigSchema(),
  })
}

export function createGroupGuardPluginConfigSchema(): Schema<StuhelperGroupGuardPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    guard: createGuardConfigSchema(),
    scheduler: createSchedulerConfigSchema(),
  })
}

export function createAdminPluginConfigSchema(): Schema<StuhelperAdminPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    admin: createAdminConfigSchema(),
  })
}
