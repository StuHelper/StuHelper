import type { Session } from 'koishi'

import { formatDuration, parseTimeString, parseUserId } from '../../utils'
import type { WarnModule } from './warn.module'

interface WarnCommandInput {
  host: WarnModule
  session: Session
  user?: unknown
  count?: number
}

interface AutoBanInput {
  host: WarnModule
  session: Session
  userId: string
  warnCount: number
  addedCount: number
}

export function registerWarnCommands(host: WarnModule): void {
  registerWarnCommand(host)
  registerClearWarnCommand(host)
  registerListWarnCommand(host)
}

function registerWarnCommand(host: WarnModule): void {
  host.registerCommand({
    name: 'warn',
    desc: '警告用户',
    args: '<user:user> [count:number]',
    permNode: 'add',
    permDesc: '添加警告记录',
    usage: '警告用户，达到阈值后自动禁言',
    examples: ['warn @用户', 'warn @用户 3'],
  }).action(async ({ session }, user, count = 1) => {
    return handleWarnCommand({ host, session, user, count })
  })
}

async function handleWarnCommand(input: WarnCommandInput): Promise<string> {
  const { host, session, user, count = 1 } = input
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
  if (!user) return '请指定要警告的用户喵！'

  const userId = parseUserId(user)
  if (!userId) return '请指定正确的用户喵！'

  const warnCount = host.addWarn(session.guildId, userId, count)
  const groupConfig = host.getGroupConfig(session.guildId)
  const warnLimit = groupConfig?.warnLimit ?? host.config.warnLimit
  if (warnLimit === 0 || warnCount >= warnLimit) {
    return executeAutoBan({ host, session, userId, warnCount, addedCount: count })
  }

  await pushWarnMessage(
    host,
    session,
    `[警告] 用户 ${userId} 在群 ${session.guildId} 被警告 ${count} 次，累计 ${warnCount} 次，未触发自动禁言`,
  )
  void host.log({ session, command: 'warn', target: userId, result: `已警告 ${count} 次，累计 ${warnCount} 次` })
  return `已警告用户 ${userId}\n本群警告：${warnCount} 次`
}

async function executeAutoBan(input: AutoBanInput): Promise<string> {
  const { host, session, userId, warnCount, addedCount } = input
  const expression = host.config.banTimes.expression.replace(/{t}/g, String(warnCount))

  try {
    const milliseconds = parseTimeString(expression)
    await session.bot.muteGuildMember(session.guildId, userId, milliseconds)
    host.recordMute(session.guildId, userId, milliseconds)
    await pushWarnMessage(
      host,
      session,
      `[警告] 用户 ${userId} 在群 ${session.guildId} 被警告 ${addedCount} 次，累计 ${warnCount} 次，触发自动禁言 ${formatDuration(milliseconds)}`,
    )
    void host.log({ session, command: 'warn', target: userId, result: `成功：已警告 ${addedCount} 次，累计 ${warnCount} 次，触发自动禁言 ${formatDuration(milliseconds)}` })
    return `已警告用户 ${userId}\n本群警告：${warnCount} 次\n已自动禁言 ${formatDuration(milliseconds)}`
  } catch (error: any) {
    await pushWarnMessage(
      host,
      session,
      `[警告] 用户 ${userId} 在群 ${session.guildId} 被警告 ${addedCount} 次，累计 ${warnCount} 次，但自动禁言失败：${error.message}`,
    )
    void host.log({ session, command: 'warn', target: userId, result: `失败：已警告 ${addedCount} 次，累计 ${warnCount} 次，但自动禁言失败` })
    return `警告已记录，但自动禁言失败：${error.message}`
  }
}

function registerClearWarnCommand(host: WarnModule): void {
  host.registerCommand({
    name: 'warn.clear',
    desc: '清除用户警告',
    args: '<user:user>',
    permNode: 'clear',
    permDesc: '清除用户的警告记录',
    examples: ['warn.clear @用户'],
  }).action(async ({ session }, user) => handleClearWarnCommand({ host, session, user }))
}

async function handleClearWarnCommand(input: WarnCommandInput): Promise<string> {
  const { host, session, user } = input
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
  if (!user) return '请指定要清除警告的用户喵！'

  const userId = parseUserId(user)
  if (!userId) return '请指定正确的用户喵！'

  const guildWarns = host.data.warns.get(session.guildId)
  if (!guildWarns || !guildWarns[userId]) return `用户 ${userId} 在本群没有警告记录`

  const oldCount = guildWarns[userId].count
  delete guildWarns[userId]
  if (Object.keys(guildWarns).length === 0) {
    host.data.warns.delete(session.guildId)
  } else {
    host.data.warns.set(session.guildId, guildWarns)
  }
  host.data.warns.flush()
  void host.log({ session, command: 'warn.clear', target: userId, result: `已清除 ${oldCount} 次警告` })
  return `已清除用户 ${userId} 在本群的 ${oldCount} 次警告`
}

function registerListWarnCommand(host: WarnModule): void {
  host.registerCommand({
    name: 'warn.list',
    desc: '查看警告列表',
    args: '[user:user]',
    permNode: 'list',
    permDesc: '查看警告记录列表',
    usage: '不指定用户则显示本群所有警告',
    examples: ['warn.list', 'warn.list @用户'],
  }).action(async ({ session }, user) => handleListWarnsCommand({ host, session, user }))
}

async function handleListWarnsCommand(input: WarnCommandInput): Promise<string> {
  const { host, session, user } = input
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

  const guildWarns = host.data.warns.get(session.guildId)
  if (user) {
    const userId = parseUserId(user)
    if (!userId) return '请指定正确的用户喵！'
    return formatUserWarns(guildWarns, userId)
  }
  if (!guildWarns || Object.keys(guildWarns).length === 0) return '本群暂无警告记录'

  const list = Object.entries(guildWarns)
    .map(([userId, record]) => ({ userId, count: record.count }))
    .sort((a, b) => b.count - a.count)
  const lines = list.slice(0, 10).map((item, index) => `${index + 1}. ${item.userId} - ${item.count} 次`)
  return `本群警告记录（前10名）：\n${lines.join('\n')}`
}

function formatUserWarns(
  guildWarns: Record<string, { count: number; timestamp: number }> | undefined,
  userId: string,
): string {
  if (!guildWarns || !guildWarns[userId]) return `用户 ${userId} 在本群没有警告记录`

  const { count, timestamp } = guildWarns[userId]
  const date = new Date(timestamp).toLocaleString('zh-CN')
  return `用户 ${userId} 警告记录：\n本群警告：${count} 次\n最后警告时间：${date}`
}

async function pushWarnMessage(host: WarnModule, session: Session, message: string): Promise<void> {
  await host.ctx.stuhelperGroupCenter.pushMessage(session.bot, message, 'warning')
}
