import { Schema } from 'koishi'

import type {
  StuhelperAdminPluginConfig,
  StuhelperAdmissionActionStreamConfig,
  StuhelperBindingPluginConfig,
  StuhelperCoreConfig,
  StuhelperGroupGuardPluginConfig,
  StuhelperPlatformConfig,
  StuhelperRetentionConfig,
  StuhelperSchedulerConfig,
} from '../types/index'

const DEFAULT_SCAN_INTERVAL_SECONDS = 300
const DEFAULT_ACTION_STREAM_RECONNECT_DELAY_SECONDS = 5
const DEFAULT_PLATFORM_BASE_URL = 'http://127.0.0.1:8080'
const DEFAULT_MESSAGE_RETENTION_DAYS = 14
const DEFAULT_EVENT_RETENTION_DAYS = 30

export function createPlatformConfigSchema(): Schema<StuhelperPlatformConfig> {
  return Schema.object({
    baseUrl: Schema.string().default(DEFAULT_PLATFORM_BASE_URL).description('StuHelper 平台 API 根地址。'),
    serviceToken: Schema.string().default('').description('机器人访问 StuHelper 平台的服务令牌。'),
  })
}

export function createSchedulerConfigSchema(): Schema<StuhelperSchedulerConfig> {
  return Schema.object({
    scanIntervalSeconds: Schema.number().min(1).default(DEFAULT_SCAN_INTERVAL_SECONDS).description('后台兜底扫描待认证成员的间隔，单位为秒。'),
  })
}

export function createAdmissionActionStreamConfigSchema(): Schema<StuhelperAdmissionActionStreamConfig> {
  return Schema.object({
    reconnectDelaySeconds: Schema.number().min(1).default(DEFAULT_ACTION_STREAM_RECONNECT_DELAY_SECONDS).description('SSE 断线后的重连延迟，单位为秒。'),
  })
}

export function createCoreConfigSchema(): Schema<StuhelperCoreConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
  })
}

export function createBindingPluginConfigSchema(): Schema<StuhelperBindingPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
  })
}

export function createRetentionConfigSchema(): Schema<StuhelperRetentionConfig> {
  return Schema.object({
    messageRetentionDays: Schema.number().min(1).default(DEFAULT_MESSAGE_RETENTION_DAYS).description('消息台账（防撤回）记录保留天数，过期记录由每日修剪任务删除。'),
    eventRetentionDays: Schema.number().min(1).default(DEFAULT_EVENT_RETENTION_DAYS).description('群管事件日志保留天数，过期记录由每日修剪任务删除。'),
  })
}

export function createGroupGuardPluginConfigSchema(): Schema<StuhelperGroupGuardPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    scheduler: createSchedulerConfigSchema(),
    actionStream: createAdmissionActionStreamConfigSchema(),
    retention: createRetentionConfigSchema(),
  })
}

export function createAdminPluginConfigSchema(): Schema<StuhelperAdminPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
  })
}
