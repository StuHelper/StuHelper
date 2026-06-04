import { Schema } from 'koishi'

import type {
  StuhelperAdminConfig,
  StuhelperAdminPluginConfig,
  StuhelperAdmissionCommandConfig,
  StuhelperAdmissionActionStreamConfig,
  StuhelperBindingConfig,
  StuhelperBindingPluginConfig,
  StuhelperCommandConfig,
  StuhelperConsoleConfig,
  StuhelperConsolePluginConfig,
  StuhelperCoreConfig,
  StuhelperFreshmanForwardConfig,
  StuhelperFunConfig,
  StuhelperGuardConfig,
  StuhelperGroupGuardPluginConfig,
  StuhelperKeywordRuleConfig,
  StuhelperModerationConfig,
  StuhelperPlatformConfig,
  StuhelperSchedulerConfig,
  StuhelperAIConfig,
} from '../types/index'

const DEFAULT_BINDING_COMMAND = '绑定'
const DEFAULT_CODE_TTL_MINUTES = 10
const DEFAULT_MUTE_DURATION_SECONDS = 600
const DEFAULT_KICK_AFTER_MINUTES = 30
const DEFAULT_SCAN_INTERVAL_SECONDS = 300
const DEFAULT_ACTION_STREAM_RECONNECT_DELAY_SECONDS = 5
const DEFAULT_REMINDER_TEMPLATE = '请先完成 StuHelper 注册、QQ 绑定与学生认证。'
const DEFAULT_PLATFORM_BASE_URL = 'http://127.0.0.1:8080'
const DEFAULT_REPEAT_THRESHOLD = 3
const DEFAULT_REPEAT_WINDOW_SIZE = 3
const DEFAULT_WARNING_EXPRESSION = 'warnings >= 3'
const DEFAULT_DICE_SIDES = 100
const DEFAULT_MUTE_LOTTERY_BASE_SECONDS = 120
const DEFAULT_MUTE_LOTTERY_MAX_SECONDS = 600
const DEFAULT_MUTE_LOTTERY_PITY_THRESHOLD = 5
const DEFAULT_MUTE_LOTTERY_PITY_SECONDS = 300
const DEFAULT_CONSOLE_TITLE = 'StuHelper 群管中心'
const DEFAULT_ADMISSION_COMMAND_AUTHORITY = 4

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
    scanIntervalSeconds: Schema.number().min(1).default(DEFAULT_SCAN_INTERVAL_SECONDS).description('后台兜底扫描待认证成员的间隔，单位为秒。'),
    fallbackScanEnabled: Schema.boolean().default(true).description('是否启用低频后台兜底扫描。主路径应使用 admission action stream。'),
  })
}

export function createAdmissionActionStreamConfigSchema(): Schema<StuhelperAdmissionActionStreamConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(true).description('是否启用后端 admission action SSE 下行流。'),
    reconnectDelaySeconds: Schema.number().min(1).default(DEFAULT_ACTION_STREAM_RECONNECT_DELAY_SECONDS).description('SSE 断线后的重连延迟，单位为秒。'),
  })
}

export function createKeywordRuleConfigSchema(): Schema<StuhelperKeywordRuleConfig> {
  return Schema.object({
    id: Schema.string().required().description('规则 ID。'),
    guildId: Schema.string().default('*').description('群号；`*` 表示全局规则。'),
    pattern: Schema.string().required().description('关键词或正则表达式。'),
    matchMode: Schema.union(['includes', 'regex']).default('includes').description('命中模式。'),
    action: Schema.union(['warn', 'delete', 'mute', 'review']).default('warn').description('命中后的动作。'),
    enabled: Schema.boolean().default(true).description('是否启用该关键词规则。'),
    muteSeconds: Schema.number().min(0).default(0).description('当动作为 mute 时使用的禁言秒数。'),
    note: Schema.string().default('').description('规则备注。'),
  })
}

export function createModerationConfigSchema(): Schema<StuhelperModerationConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(true).description('是否启用消息风控监听。'),
    repeatThreshold: Schema.number().min(2).default(DEFAULT_REPEAT_THRESHOLD).description('复读命中阈值。'),
    repeatWindowSize: Schema.number().min(2).default(DEFAULT_REPEAT_WINDOW_SIZE).description('复读检测窗口大小。'),
    warningThresholdExpression: Schema.string().default(DEFAULT_WARNING_EXPRESSION).description('警告升级表达式，例如 `warnings >= 3`。'),
    defaultMuteSeconds: Schema.number().min(1).default(DEFAULT_MUTE_DURATION_SECONDS).description('自动处罚默认禁言时长。'),
    antiRecallNotify: Schema.boolean().default(true).description('是否在检测到撤回后推送提示。'),
    keywordRules: Schema.array(createKeywordRuleConfigSchema()).default([]).description('启动时注入的默认关键词规则。'),
  })
}

export function createFunConfigSchema(): Schema<StuhelperFunConfig> {
  return Schema.object({
    diceSides: Schema.number().min(2).default(DEFAULT_DICE_SIDES).description('骰子命令默认面数。'),
    muteLotteryBaseSeconds: Schema.number().min(1).default(DEFAULT_MUTE_LOTTERY_BASE_SECONDS).description('抽禁言基础时长。'),
    muteLotteryMaxSeconds: Schema.number().min(1).default(DEFAULT_MUTE_LOTTERY_MAX_SECONDS).description('抽禁言最大时长。'),
    muteLotteryPityThreshold: Schema.number().min(1).default(DEFAULT_MUTE_LOTTERY_PITY_THRESHOLD).description('触发保底前需要累计的抽卡次数。'),
    muteLotteryPitySeconds: Schema.number().min(1).default(DEFAULT_MUTE_LOTTERY_PITY_SECONDS).description('保底时至少获得的禁言秒数。'),
  })
}

export function createCommandConfigSchema(): Schema<StuhelperCommandConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(true).description('是否注册举报、骰子、抽禁言等公开命令。'),
  })
}

export function createAdmissionCommandConfigSchema(): Schema<StuhelperAdmissionCommandConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(true).description('是否注册入群认证管理员命令。'),
    minAuthority: Schema.number().min(0).default(DEFAULT_ADMISSION_COMMAND_AUTHORITY).description('执行入群认证管理员命令所需的最低 Koishi 权限等级。'),
    operatorQQIDs: Schema.array(Schema.string()).default([]).description('允许执行入群认证管理员命令的 QQ 号白名单。'),
  })
}

export function createFreshmanForwardConfigSchema(): Schema<StuhelperFreshmanForwardConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(true).description('是否扫描并转发待审核新生原始材料到管理群。'),
  })
}

export function createConsoleConfigSchema(): Schema<StuhelperConsoleConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(true).description('是否启用自定义 Koishi Console 页面。'),
    title: Schema.string().default(DEFAULT_CONSOLE_TITLE).description('群管中心页面标题。'),
  })
}

export function createAIConfigSchema(): Schema<StuhelperAIConfig> {
  return Schema.object({
    enabled: Schema.boolean().default(false).description('是否启用 AI 举报审核。'),
    endpoint: Schema.string().default('').description('AI 审核 HTTP 接口地址。'),
    apiKey: Schema.string().default('').description('AI 审核接口令牌。'),
    model: Schema.string().default('').description('AI 审核模型标识。'),
  })
}

export function createCoreConfigSchema(): Schema<StuhelperCoreConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    guard: createGuardConfigSchema(),
    console: createConsoleConfigSchema(),
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
    actionStream: createAdmissionActionStreamConfigSchema(),
    moderation: createModerationConfigSchema(),
    fun: createFunConfigSchema(),
    ai: createAIConfigSchema(),
    commands: createCommandConfigSchema(),
    admissionCommands: createAdmissionCommandConfigSchema(),
    freshmanForward: createFreshmanForwardConfigSchema(),
  })
}

export function createAdminPluginConfigSchema(): Schema<StuhelperAdminPluginConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    admin: createAdminConfigSchema(),
    moderation: createModerationConfigSchema(),
    fun: createFunConfigSchema(),
  })
}

export function createConsolePluginConfigSchema(): Schema<StuhelperConsolePluginConfig> {
  return Schema.object({
    console: createConsoleConfigSchema(),
    moderation: createModerationConfigSchema(),
  })
}
