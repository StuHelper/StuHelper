import type { Session } from 'koishi'

import { formatDuration, parseTimeString } from '../../utils'
import { resolveCommandUserId, resolveTargetUserId, splitTargetArgs } from './member-manage-input'
import type { OrderManageModule } from './orderManage.module'
import { commandErrorMessage } from './command-error-message'

const SHORT_MUTE_MS = 600_000

interface TargetInput {
  userId: string | null
  groupId?: string
  duration?: string
  hasEnoughArgs: boolean
}

export function registerOrderBanCommands(host: OrderManageModule): void {
  registerBanCommand(host)
  registerStopCommand(host)
  registerUnbanCommand(host)
}

function registerBanCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'ban',
    desc: '禁言用户',
    args: '<input:text>',
    permNode: 'ban',
    permDesc: '禁言群成员',
    usage: '格式：ban <用户> <时长> [群号]',
    examples: ['ban @用户 1h', 'ban 123456789 30min'],
  })
    .example('ban @用户 1h')
    .example('ban 123456789 1h')
    .example('ban @用户 1h 群号')
    .action(async ({ session }, input) => handleBanCommand(host, session, input))
}

async function handleBanCommand(host: OrderManageModule, session: Session, input: string): Promise<string> {
  if (!input) {
    host.logCommand({ session, command: 'ban', target: 'none', result: '失败：缺少必要参数', success: false })
    return '喵呜...格式：ban &lt;用户> &lt;时长> [群号]'
  }

  const parsed = parseBanInput(session, input)
  if (!parsed.hasEnoughArgs) {
    host.logCommand({ session, command: 'ban', target: 'none', result: '失败：缺少必要参数', success: false })
    return '喵呜...格式：ban &lt;用户> &lt;时长> [群号]'
  }
  if (!parsed.duration) {
    host.logCommand({ session, command: 'ban', target: parsed.userId, result: '失败：未指定禁言时长', success: false })
    return parsed.userId ? '喵呜...请告诉我要禁言多久呀~' : '喵呜...格式：ban &lt;用户> &lt;时长> [群号]'
  }
  if (!parsed.userId) {
    host.logCommand({ session, command: 'ban', target: 'none', result: '失败：无法读取目标用户', success: false })
    return '喵呜...请输入正确的用户（@或QQ号）'
  }
  if (!parsed.groupId) {
    host.logCommand({ session, command: 'ban', target: parsed.userId, result: '失败：缺少群号', success: false })
    return '喵呜...请在群聊中执行，或显式传入群号'
  }

  try {
    const milliseconds = parseTimeString(parsed.duration)
    await session.bot.muteGuildMember(parsed.groupId, parsed.userId, milliseconds)
    host.recordMute(parsed.groupId, parsed.userId, milliseconds)
    const timeStr = formatDuration(milliseconds)
    host.logCommand({ session, command: 'ban', target: parsed.userId, result: `成功：已禁言 ${timeStr}，群号：${parsed.groupId}` })
    return `已经把 ${parsed.userId} 禁言 ${parsed.duration} (${timeStr}) 啦喵~`
  } catch (error) {
    const message = commandErrorMessage(error)
    host.logCommand({ session, command: 'ban', target: parsed.userId, result: '失败：未知错误', success: false })
    return `喵呜...禁言失败了：${message}`
  }
}

function parseBanInput(session: Session, rawInput: string): TargetInput {
  let input = rawInput
  const quoteContent = String(session.quote?.content ?? '')
  if (quoteContent && input.endsWith(quoteContent)) {
    input = input.slice(0, input.length - quoteContent.length).trim()
  }
  const args = splitTargetArgs(input)
  const [target, duration, groupId] = args
  return {
    userId: resolveTargetUserId(target),
    duration,
    groupId: groupId || session.guildId,
    hasEnoughArgs: args.length >= 2,
  }
}

function registerStopCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'stop',
    desc: '短期禁言',
    args: '<user:user>',
    permNode: 'stop',
    permDesc: '短期禁言（10分钟）',
    usage: '固定10分钟的短期禁言',
    examples: ['stop @用户'],
  }).action(async ({ session }, user) => handleStopCommand(host, session, user))
}

async function handleStopCommand(host: OrderManageModule, session: Session, user: unknown): Promise<string> {
  if (!user) return '请指定用户'
  if (!session.guildId) return '请在群聊中执行该命令。'

  const userId = resolveCommandUserId(user)
  if (!userId) return '请指定正确的用户'
  const guildMutes = host.data.mutes.getAll()[session.guildId] || {}
  const lastMute = guildMutes[userId] || { startTime: 0, duration: 0 }
  if (lastMute.startTime + lastMute.duration > Date.now()) {
    host.logCommand({ session, command: 'stop', target: userId, result: '失败：已在禁言中', success: false })
    return `喵呜...${userId} 已经处于禁言状态啦，不需要短期禁言喵~`
  }

  try {
    await session.bot.muteGuildMember(session.guildId, userId, SHORT_MUTE_MS)
    host.recordMute(session.guildId, userId, SHORT_MUTE_MS)
    host.logCommand({ session, command: 'stop', target: userId, result: `成功：已短期禁言，群号 ${session.guildId}` })
    return `已将 ${userId} 短期禁言啦喵~`
  } catch (error) {
    const message = commandErrorMessage(error)
    host.logCommand({ session, command: 'stop', target: userId, result: '失败：未知错误', success: false })
    return `喵呜...短期禁言失败了：${message}`
  }
}

function registerUnbanCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'unban',
    desc: '解除用户禁言',
    args: '<input:text>',
    permNode: 'unban',
    permDesc: '解除禁言',
    usage: '格式：unban <用户> [群号]',
    examples: ['unban @用户', 'unban 123456789'],
  })
    .example('unban @用户')
    .example('unban 123456789')
    .example('unban @用户 群号')
    .action(async ({ session }, input) => handleUnbanCommand(host, session, input))
}

async function handleUnbanCommand(host: OrderManageModule, session: Session, input: string): Promise<string> {
  const [target, groupId] = splitTargetArgs(input)
  const userId = resolveTargetUserId(target)
  if (!userId) {
    host.logCommand({ session, command: 'unban', target: 'none', result: '失败：无法读取目标用户', success: false })
    return '喵呜...请输入正确的用户（@或QQ号）'
  }

  const targetGroup = groupId || session.guildId
  if (!targetGroup) {
    host.logCommand({ session, command: 'unban', target: userId, result: '失败：缺少群号', success: false })
    return '喵呜...请在群聊中执行，或显式传入群号'
  }

  try {
    await session.bot.muteGuildMember(targetGroup, userId, 0)
    host.recordMute(targetGroup, userId, 0)
    host.logCommand({ session, command: 'unban', target: userId, result: `成功：已解除禁言，群号 ${targetGroup}` })
    return `已经把 ${userId} 的禁言解除啦喵！开心~`
  } catch (error) {
    const message = commandErrorMessage(error)
    host.logCommand({ session, command: 'unban', target: userId, result: '失败：未知错误', success: false })
    return `喵呜...解除禁言失败了：${message}`
  }
}
