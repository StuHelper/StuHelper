import { Schema } from 'koishi'

import type {
  StuhelperAdminConfig,
  StuhelperAdminPluginConfig,
  StuhelperAdmissionCommandConfig,
  StuhelperAdmissionActionStreamConfig,
  StuhelperBindingMessageConfig,
  StuhelperBindingConfig,
  StuhelperBindingPluginConfig,
  StuhelperCommandConfig,
  StuhelperConsoleConfig,
  StuhelperConsolePluginConfig,
  StuhelperCoreConfig,
  StuhelperFreshmanForwardConfig,
  StuhelperFunConfig,
  StuhelperGuardConfig,
  StuhelperGroupGuardMessageConfig,
  StuhelperGroupGuardPluginConfig,
  StuhelperKeywordRuleConfig,
  StuhelperModerationConfig,
  StuhelperPlatformConfig,
  StuhelperSchedulerConfig,
  StuhelperAIConfig,
} from '../types/index'
import {
  DEFAULT_BINDING_MESSAGES,
  DEFAULT_GROUP_GUARD_MESSAGES,
} from '../message-template'

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
    messages: createBindingMessageConfigSchema().description('绑定命令反馈文案。模板变量用 `{变量名}` 表示。'),
  })
}

export function createBindingMessageConfigSchema(): Schema<StuhelperBindingMessageConfig> {
  return Schema.object({
    directOnly: messageTemplate(DEFAULT_BINDING_MESSAGES.directOnly, '群聊内执行绑定命令时返回。可用变量：无。'),
    missingCode: messageTemplate(DEFAULT_BINDING_MESSAGES.missingCode, '未提供绑定码时返回。可用变量：`{command}`。'),
    successVerified: messageTemplate(DEFAULT_BINDING_MESSAGES.successVerified, '绑定成功且账号已完成学生认证时返回。可用变量：无。'),
    successUnverified: messageTemplate(DEFAULT_BINDING_MESSAGES.successUnverified, '绑定成功但账号未完成学生认证时返回。可用变量：无。'),
    unavailable: messageTemplate(DEFAULT_BINDING_MESSAGES.unavailable, '平台不可用或未知错误时返回。可用变量：无。'),
    invalidCode: messageTemplate(DEFAULT_BINDING_MESSAGES.invalidCode, '绑定码无效或过期时返回。可用变量：无。'),
    unauthorized: messageTemplate(DEFAULT_BINDING_MESSAGES.unauthorized, '机器人服务鉴权失败时返回。可用变量：无。'),
    conflict: messageTemplate(DEFAULT_BINDING_MESSAGES.conflict, 'QQ 或 StuHelper 账号已绑定其他对象时返回。可用变量：无。'),
    notConfigured: messageTemplate(DEFAULT_BINDING_MESSAGES.notConfigured, '后端机器人接口未配置时返回。可用变量：无。'),
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

export function createGroupGuardMessageConfigSchema(): Schema<StuhelperGroupGuardMessageConfig> {
  return Schema.object({
    admissionReminder: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionReminder, '认证提醒主文案。可用变量：`{at}`、`{memberId}`、`{minutes}`、`{authURL}`、`{timeoutLine}`。'),
    admissionTimeoutNormal: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionTimeoutNormal, '首次或普通超时说明。可用变量：`{failureCount}`、`{remainingRetryCount}`。'),
    admissionTimeoutWithFailures: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionTimeoutWithFailures, '已有未认证次数但本次不会拉黑时的超时说明。可用变量：`{failureCount}`、`{remainingRetryCount}`。'),
    admissionTimeoutBlacklist: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionTimeoutBlacklist, '本次超时会拉黑时的超时说明。可用变量：`{failureCount}`、`{remainingRetryCount}`。'),
    backendPendingReminder: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.backendPendingReminder, '后端暂不可用、链接暂未创建时的兜底提醒。可用变量：`{at}`、`{memberId}`、`{reminderTemplate}`。'),
    admissionReleaseCompleted: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionReleaseCompleted, '认证通过并自动解除禁言后的群内提示。留空表示不发送。可用变量：`{at}`、`{memberId}`。'),
    admissionKickTimeout: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionKickTimeout, '认证超时移出群聊前提示。可用变量：`{at}`、`{memberId}`。'),
    admissionBlacklistKick: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionBlacklistKick, '达到失败次数上限并拉黑前提示。可用变量：`{at}`、`{memberId}`。'),
    antiRecallNotify: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.antiRecallNotify, '检测到撤回消息时的群内提示。可用变量：`{at}`、`{memberId}`、`{content}`。'),
    moderationMuteNotice: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.moderationMuteNotice, '自动禁言提示。留空表示只执行禁言不发提示。可用变量：`{at}`、`{memberId}`、`{reason}`、`{seconds}`。'),
    moderationUnmuteNotice: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.moderationUnmuteNotice, '解除禁言提示。留空表示只执行解禁不发提示。可用变量：`{at}`、`{memberId}`、`{reason}`。'),
    moderationKickNotice: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.moderationKickNotice, '自动移出群聊提示。留空表示只执行踢出不发提示。可用变量：`{at}`、`{memberId}`、`{reason}`。'),
    freshmanForwardSummary: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.freshmanForwardSummary, '新生材料转发到审核群时的摘要。可用变量：`{applicationId}`、`{applicantName}`、`{schoolName}`、`{departmentOrMajor}`、`{qqID}`、`{materialType}`、`{provisionalExpiresAt}`。'),
    publicReportMissingArgs: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.publicReportMissingArgs, '举报命令缺少参数时返回。'),
    publicCommandsDisabled: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.publicCommandsDisabled, '公开命令被 WebUI 关闭时返回。'),
    muteLotteryGroupOnly: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.muteLotteryGroupOnly, '抽禁言命令不在群聊中执行时返回。'),
    commandAccessDenied: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.commandAccessDenied, '命令权限不足时返回。'),
    diceResult: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.diceResult, '骰子结果。可用变量：`{memberId}`、`{sides}`、`{result}`。'),
    muteLotteryResult: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.muteLotteryResult, '普通抽禁言结果。可用变量：`{memberId}`、`{seconds}`。'),
    muteLotteryPityResult: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.muteLotteryPityResult, '保底抽禁言结果。可用变量：`{memberId}`、`{seconds}`。'),
    reportGroupOnly: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportGroupOnly, '举报命令不在群聊中执行时返回。'),
    reportRecordedAIUnavailable: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportRecordedAIUnavailable, '举报已记录但未启用 AI 审核时返回。'),
    reportAIReviewFailed: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportAIReviewFailed, '举报 AI 审核失败时返回。'),
    reportHighRisk: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportHighRisk, 'AI 判定高风险时返回。'),
    reportMediumRisk: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportMediumRisk, 'AI 判定中风险时返回。'),
    reportLowRisk: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportLowRisk, 'AI 判定低风险时返回。'),
    reportNoAction: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.reportNoAction, 'AI 未判定可执行动作时返回。'),
    admissionCommandGroupOnly: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandGroupOnly, '入群认证管理员命令不在群聊中执行时返回。'),
    admissionCommandsDisabled: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandsDisabled, '入群认证管理员命令被 WebUI 关闭时返回。'),
    admissionCommandMissingQQ: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandMissingQQ, '管理员命令缺少 QQ 号时返回。'),
    admissionCommandMissingOperator: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandMissingOperator, '无法识别操作者 QQ 时返回。'),
    admissionCommandUnsupportedPlatform: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandUnsupportedPlatform, '当前平台不支持入群认证时返回。'),
    admissionCommandPolicyDisabled: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandPolicyDisabled, '当前群未启用入群认证时返回。'),
    admissionCommandNotFound: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandNotFound, '找不到入群认证记录时返回。'),
    admissionCommandInvalidState: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandInvalidState, '当前认证状态不允许操作时返回。'),
    admissionCommandUnauthorized: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandUnauthorized, '机器人凭据无权限时返回。'),
    admissionCommandFailed: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandFailed, '管理员命令未知错误时返回。可用变量：`{error}`。'),
    admissionCommandPlatformError: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandPlatformError, '平台接口异常时返回。可用变量：`{status}`、`{message}`。'),
    admissionCommandMissingResendURL: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandMissingResendURL, '后端未返回可重发认证链接时返回。'),
    admissionCommandStaleRecord: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionCommandStaleRecord, '本地记录被其他任务处理时返回。'),
    admissionSkipSuccess: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionSkipSuccess, '跳过入群认证成功时返回。可用变量：`{at}`、`{qqID}`。'),
    admissionAlreadyVerifiedRegenerate: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionAlreadyVerifiedRegenerate, '重新生成链接时发现成员已认证通过的返回文案。可用变量：`{at}`、`{qqID}`。'),
    admissionResetFailureCountSuccess: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionResetFailureCountSuccess, '清空未认证次数成功时返回。可用变量：`{qqID}`、`{previousFailureCount}`。'),
    admissionReleaseBlacklistNotFound: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionReleaseBlacklistNotFound, '解除入群拉黑但未找到记录时返回。可用变量：`{qqID}`。'),
    admissionReleaseBlacklistSuccess: messageTemplate(DEFAULT_GROUP_GUARD_MESSAGES.admissionReleaseBlacklistSuccess, '解除入群拉黑成功时返回。可用变量：`{qqID}`。'),
  }).description('群管插件所有用户可见提示文案。模板变量使用 `{变量名}`；可发送的提示留空表示不发送。')
}

export function createCoreConfigSchema(): Schema<StuhelperCoreConfig> {
  return Schema.object({
    platform: createPlatformConfigSchema(),
    guard: createGuardConfigSchema(),
    console: createConsoleConfigSchema(),
    runtimeModules: Schema.object({
      enabled: Schema.boolean().default(true).description('是否启用 stuhelper-core 旧群管运行时模块。生产只需要 WebUI/API 时应关闭，避免注册旧命令与既有插件冲突。'),
    }).description('stuhelper-core 运行时模块开关。'),
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
    messages: createGroupGuardMessageConfigSchema(),
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

function messageTemplate(defaultValue: string, description: string) {
  return Schema.string().default(defaultValue).description(`${description} 推荐默认值：${defaultValue || '留空'}`)
}
