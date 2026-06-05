import type { Session } from 'koishi'

import type { ReportConfig } from '../../types'
import type { ReportModule } from './report.module'

const DEFAULT_CONTEXT_SIZE = 5
const MIN_CONTEXT_SIZE = 1
const MAX_CONTEXT_SIZE = 20

interface ReportConfigInput {
  readonly host: ReportModule
  readonly session: Session
  readonly options: ReportConfigOptions
}

interface ReportConfigOptions {
  readonly enabled?: boolean
  readonly auto?: boolean
  readonly authority?: number
  readonly context?: boolean
  readonly 'context-size'?: number
  readonly guild?: string
}

export function registerReportConfigCommand(host: ReportModule): void {
  host.registerCommand({
    name: 'report-config',
    desc: '配置举报功能',
    permNode: 'report-config',
    permDesc: '配置举报功能',
    usage: '配置举报功能的启用、自动处理、上下文等选项',
  })
    .option('enabled', '-e <enabled:boolean> 是否启用举报功能')
    .option('auto', '-a <auto:boolean> 是否自动处理违规')
    .option('authority', '-auth <auth:number> 设置举报功能权限等级')
    .option('context', '-c <context:boolean> 是否包含群聊上下文')
    .option('context-size', '-cs <size:number> 上下文消息数量')
    .option('guild', '-g <guildId:string> 配置指定群聊')
    .action(async ({ session, options }) => {
      if (!session) return '无法读取当前会话'
      return handleReportConfigCommand({ host, session, options })
    })
}

async function handleReportConfigCommand(input: ReportConfigInput): Promise<string> {
  const guildId = input.options.guild || input.session.guildId
  if (!guildId) return '请在群聊中使用此命令或使用 -g 参数指定群号'

  const currentReport = { ...input.host.config.report }
  const result = applyGuildReportConfig(input, currentReport, guildId)
  if (typeof result === 'string') return result

  if (!result.hasChanges) return result.messages.join('\n')

  await input.host.ctx.stuhelperGroupCenter.settings.update({ report: currentReport })
  await input.host.logCommand({
    session: input.session,
    command: 'report-config',
    target: guildId,
    details: '已更新举报功能配置',
  })
  return `举报功能配置已更新\n${result.messages.join('\n')}`
}

function applyGuildReportConfig(
  input: ReportConfigInput,
  currentReport: ReportConfig,
  guildId: string,
): { hasChanges: boolean; messages: string[] } | string {
  const messages = [`群 ${guildId} 的举报功能配置：`]
  currentReport.guildConfigs = currentReport.guildConfigs || {}
  currentReport.guildConfigs[guildId] = currentReport.guildConfigs[guildId] || {
    enabled: true,
    includeContext: false,
    contextSize: DEFAULT_CONTEXT_SIZE,
    autoProcess: true,
  }

  const guildConfig = currentReport.guildConfigs[guildId]
  const sizeError = applyGuildOptions(input, guildConfig)
  if (sizeError) return sizeError

  messages.push(`状态: ${guildConfig.enabled ? '已启用' : '已禁用'}`)
  messages.push(`自动处理: ${guildConfig.autoProcess ? '已启用' : '已禁用'}`)
  messages.push(`包含上下文: ${guildConfig.includeContext ? '已启用' : '已禁用'}`)
  messages.push(`上下文消息数量: ${guildConfig.contextSize || DEFAULT_CONTEXT_SIZE}`)
  return { hasChanges: hasGuildConfigChanges(input.options), messages }
}

function applyGuildOptions(input: ReportConfigInput, guildConfig: NonNullable<ReportConfig['guildConfigs']>[string]): string | null {
  if (input.options.enabled !== undefined) guildConfig.enabled = input.options.enabled
  if (input.options.auto !== undefined) guildConfig.autoProcess = input.options.auto
  if (input.options.context !== undefined) guildConfig.includeContext = input.options.context
  if (input.options['context-size'] === undefined) return null

  const size = input.options['context-size']
  if (size < MIN_CONTEXT_SIZE || size > MAX_CONTEXT_SIZE) return '上下文消息数量必须在1-20之间'
  guildConfig.contextSize = size
  return null
}

function hasGuildConfigChanges(options: ReportConfigOptions): boolean {
  return options.enabled !== undefined ||
    options.auto !== undefined ||
    options.context !== undefined ||
    options['context-size'] !== undefined
}
