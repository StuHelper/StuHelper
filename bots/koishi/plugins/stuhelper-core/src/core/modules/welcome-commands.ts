import type { Session } from 'koishi'

import type { GroupConfig } from '../../types'
import type { WelcomeModule } from './welcome.module'

const DEFAULT_WELCOME_CONFIG: GroupConfig = {
  keywords: [],
  approvalKeywords: [],
  welcomeMsg: '',
  goodbyeMsg: '',
  auto: 'false',
  reject: '答案错误，请重新申请',
  levelLimit: 0,
  leaveCooldown: 0,
}

interface WelcomeCommandOptions {
  s?: string
  r?: boolean
  t?: boolean
  l?: string
  j?: string
}

interface GoodbyeCommandOptions {
  s?: string
  r?: boolean
  t?: boolean
}

interface CommandContext {
  host: WelcomeModule
  session: Session
  groupConfig: GroupConfig
  allConfigs: Record<string, GroupConfig>
}

export function registerWelcomeCommands(host: WelcomeModule): void {
  host.registerCommand({
    name: 'welcome',
    desc: '入群欢迎语管理',
    permNode: 'welcome',
    permDesc: '管理入群欢迎语',
    usage: '-s 设置欢迎语，-r 移除，-t 测试，-l 设置等级限制，-j 设置退群冷却',
  })
    .option('s', '-s <消息> 设置欢迎语')
    .option('r', '-r 移除欢迎语')
    .option('t', '-t 测试当前欢迎语')
    .option('l', '-l <等级> 设置等级限制')
    .option('j', '-j <天数> 设置退群冷却天数')
    .action(async ({ session, options }) => handleWelcomeCommand(host, session, options))

  host.registerCommand({
    name: 'goodbye',
    desc: '退群欢送语管理',
    permNode: 'goodbye',
    permDesc: '管理退群欢送语',
    usage: '-s 设置欢送语，-r 移除，-t 测试',
  })
    .option('s', '-s <消息> 设置欢送语')
    .option('r', '-r 移除欢送语')
    .option('t', '-t 测试当前欢送语')
    .action(async ({ session, options }) => handleGoodbyeCommand(host, session, options))
}

async function handleWelcomeCommand(
  host: WelcomeModule,
  session: Session,
  options: WelcomeCommandOptions,
): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

  const allConfigs = host.getGroupConfigs()
  const groupConfig = allConfigs[session.guildId] || DEFAULT_WELCOME_CONFIG
  const context = { host, session, groupConfig, allConfigs }
  const numericResult = applyWelcomeNumericOption(context, options)
  if (numericResult) return numericResult

  if (options.s) return setWelcomeMessage(context, options.s)
  if (options.r) return removeWelcomeMessage(context)
  if (options.t) {
    const msg = groupConfig.welcomeMsg || host.config.defaultWelcome
    return msg ? host.formatWelcomeMessage(msg, session.userId, session.guildId) : '未设置欢迎语'
  }

  return formatWelcomeStatus(groupConfig)
}

function applyWelcomeNumericOption(
  context: CommandContext,
  options: WelcomeCommandOptions,
): string | null {
  if (options.l !== undefined) {
    const level = parseNonNegativeInteger(options.l)
    if (level === null) return '等级限制必须是非负整数喵~'
    context.groupConfig.levelLimit = level
    saveGroupConfig(context)
    void logWelcomeChange(context, `已设置等级限制：${level}级`)
    return `已经设置好等级限制为${level}级啦喵~`
  }

  if (options.j === undefined) return null
  const days = parseNonNegativeInteger(options.j)
  if (days === null) return '冷却天数必须是非负整数喵~'
  context.groupConfig.leaveCooldown = days
  saveGroupConfig(context)
  void logWelcomeChange(context, `已设置退群冷却：${days}天`)
  return `已经设置好退群冷却为${days}天啦喵~`
}

function setWelcomeMessage(context: CommandContext, message: string): string {
  context.groupConfig.welcomeMsg = message
  context.groupConfig.welcomeEnabled = true
  saveGroupConfig(context)
  void logWelcomeChange(context, `已设置欢迎语：${message}`)
  return `已经设置好欢迎语啦喵，要不要用 -t 试试看效果呀？`
}

function removeWelcomeMessage(context: CommandContext): string {
  context.groupConfig.welcomeMsg = ''
  context.groupConfig.welcomeEnabled = false
  saveGroupConfig(context)
  void context.host.log({
    session: context.session,
    command: 'welcome',
    target: 'remove',
    result: '已移除欢迎语',
  })
  return `欢迎语已经被我吃掉啦喵~`
}

async function handleGoodbyeCommand(
  host: WelcomeModule,
  session: Session,
  options: GoodbyeCommandOptions,
): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

  const allConfigs = host.getGroupConfigs()
  const groupConfig = allConfigs[session.guildId] || DEFAULT_WELCOME_CONFIG
  const context = { host, session, groupConfig, allConfigs }
  if (options.s) return setGoodbyeMessage(context, options.s)
  if (options.r) return removeGoodbyeMessage(context)
  if (options.t) {
    const msg = groupConfig.goodbyeMsg || host.config.defaultGoodbye
    return msg ? host.formatWelcomeMessage(msg, session.userId, session.guildId) : '未设置欢送语'
  }

  return formatGoodbyeStatus(groupConfig)
}

function setGoodbyeMessage(context: CommandContext, message: string): string {
  context.groupConfig.goodbyeMsg = message
  context.groupConfig.goodbyeEnabled = true
  saveGroupConfig(context)
  void context.host.log({
    session: context.session,
    command: 'goodbye',
    target: 'set',
    result: `已设置欢送语：${message}`,
  })
  return `已经设置好欢送语啦喵，要不要用 -t 试试看效果呀？`
}

function removeGoodbyeMessage(context: CommandContext): string {
  context.groupConfig.goodbyeMsg = ''
  context.groupConfig.goodbyeEnabled = false
  saveGroupConfig(context)
  void context.host.log({
    session: context.session,
    command: 'goodbye',
    target: 'remove',
    result: '已移除欢送语',
  })
  return `欢送语已经被我吃掉啦喵~`
}

function saveGroupConfig(context: CommandContext): void {
  context.allConfigs[context.session.guildId] = context.groupConfig
  context.host.setGroupConfigs(context.allConfigs)
}

function logWelcomeChange(context: CommandContext, result: string): Promise<void> {
  return context.host.log({
    session: context.session,
    command: 'welcome',
    target: 'set',
    result,
  })
}

function parseNonNegativeInteger(value: string): number | null {
  const result = parseInt(value)
  return isNaN(result) || result < 0 ? null : result
}

function formatWelcomeStatus(groupConfig: GroupConfig): string {
  const currentMsg = groupConfig.welcomeMsg
  const currentLevelLimit = groupConfig.levelLimit || 0
  const currentLeaveCooldown = groupConfig.leaveCooldown || 0

  return `当前欢迎语：${currentMsg || '未设置'}
当前等级限制：${currentLevelLimit}级
当前退群冷却：${currentLeaveCooldown}天

可用变量：
{at} - @新成员
{user} - 新成员QQ号
{group} - 群号

使用方法：
welcome -s <欢迎语>  设置欢迎语
welcome -r  移除欢迎语
welcome -t  测试当前欢迎语
welcome -l <等级>  设置等级限制（0表示不限制）
welcome -j <天数>  设置退群冷却天数（0表示不限制）`
}

function formatGoodbyeStatus(groupConfig: GroupConfig): string {
  const currentMsg = groupConfig.goodbyeMsg

  return `当前欢送语：${currentMsg || '未设置'}

可用变量：
{at} - @退群成员
{user} - 退群成员QQ号
{group} - 群号

使用方法：
goodbye -s <欢送语>  设置欢送语
goodbye -r  移除欢送语
goodbye -t  测试当前欢送语`
}
