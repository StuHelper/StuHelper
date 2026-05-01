import type { Session } from 'koishi'

import type { MuteRecord } from '../../types'
import { formatDuration, parseUserId } from '../../utils'
import type { OrderManageModule } from './orderManage.module'

interface NicknameCommandInput {
  host: OrderManageModule
  session: Session
  user: unknown
  nickname?: string
  group?: string
}

export function registerOrderGroupCommands(host: OrderManageModule): void {
  registerWholeBanCommand(host, true)
  registerWholeBanCommand(host, false)
  registerBanListCommand(host)
  registerNicknameCommand(host)
}

function registerWholeBanCommand(host: OrderManageModule, enabled: boolean): void {
  const commandName = enabled ? 'ban-all' : 'unban-all'
  host.registerCommand({
    name: commandName,
    desc: enabled ? '全体禁言' : '解除全体禁言',
    permNode: commandName,
    permDesc: enabled ? '开启全体禁言' : '解除全体禁言',
    usage: enabled ? '开启全群禁言模式' : '关闭全群禁言模式',
  }).action(async ({ session }) => handleWholeBanCommand(host, session, enabled))
}

async function handleWholeBanCommand(host: OrderManageModule, session: Session, enabled: boolean): Promise<string> {
  const commandName = enabled ? 'ban-all' : 'unban-all'
  if (!session.guildId) {
    host.logCommand(session, commandName, 'none', '失败：缺少群号', false)
    return '喵呜...这个命令只能在群里用喵~'
  }

  try {
    await session.bot.muteChannel(session.channelId || session.guildId, session.guildId, enabled)
    host.logCommand(session, commandName, session.guildId, enabled
      ? `成功：已开启全体禁言，群号 ${session.guildId}`
      : `成功：已解除全体禁言，群号 ${session.guildId}`)
    return enabled ? '喵呜...全体禁言开启啦，大家都要乖乖的~' : '全体禁言解除啦喵，可以开心聊天啦~'
  } catch (error) {
    host.logCommand(session, commandName, session.guildId, `失败：未知错误`, false)
    return `出错啦喵...${error}`
  }
}

function registerBanListCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'ban-list',
    desc: '查询当前禁言名单',
    permNode: 'ban-list',
    permDesc: '查询禁言名单',
    usage: '显示当前群内所有被禁言的成员',
  }).action(async ({ session }) => handleBanListCommand(host, session))
}

function handleBanListCommand(host: OrderManageModule, session: Session): string {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵~'

  const currentMutes = host.data.mutes.getAll()[session.guildId] || {}
  const formatMutes = Object.entries(currentMutes)
    .filter(([, data]) => isActiveMute(data as MuteRecord))
    .map(([userId, data]) => formatMuteEntry(userId, data as MuteRecord))
    .join('\n')

  return formatMutes ? `当前禁言名单：\n${formatMutes}` : '当前没有被禁言的成员喵~'
}

function isActiveMute(muteData: MuteRecord): boolean {
  return !muteData.leftGroup && Date.now() - muteData.startTime < muteData.duration
}

function formatMuteEntry(userId: string, muteData: MuteRecord): string {
  const remainingTime = muteData.duration - (Date.now() - muteData.startTime)
  return `用户 ${userId}：剩余 ${formatDuration(remainingTime)}`
}

function registerNicknameCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'nickname',
    desc: '设置用户昵称',
    args: '<user:user> <nickname:string> <group:string>',
    permNode: 'nickname',
    permDesc: '设置群成员昵称',
    usage: '设置指定用户的群名片，不填昵称则清除',
    examples: ['nickname @用户 小猫咪'],
  })
    .example('nickname 123456789 小猫咪')
    .action(async ({ session }, user, nickname, group) => handleNicknameCommand({
      host,
      session,
      user,
      nickname,
      group,
    }))
}

async function handleNicknameCommand(input: NicknameCommandInput): Promise<string> {
  const { host, session, user, nickname, group } = input
  if (!user) return '喵呜...请指定用户喵~'

  const userId = resolveUserId(user)
  const guildId = group || session.guildId
  if (!userId) return '喵呜...请指定正确的用户喵~'
  if (!guildId) return '喵呜...请在群聊中执行，或显式传入群号喵~'

  try {
    if (nickname) {
      await session.bot.internal.setGroupCard(guildId, userId, nickname)
      host.logCommand(session, 'nickname', userId, `成功：已设置昵称为 ${nickname}, 群号 ${guildId}`)
      return `已将 ${userId} 的昵称设置为 "${nickname}" 喵~`
    }

    await session.bot.internal.setGroupCard(guildId, userId)
    host.logCommand(session, 'nickname', userId, `成功：已清除昵称, 群号 ${guildId}`)
    return `已将 ${userId} 的昵称清除喵~`
  } catch (error) {
    host.logCommand(session, 'nickname', userId, `失败：未知错误`, false)
    return `喵呜...设置昵称失败了：${error.message}`
  }
}

function resolveUserId(user: unknown): string {
  const raw = String(user || '').trim()
  if (!raw) return ''
  const [, platformUserId] = raw.split(':')
  return platformUserId || parseUserId(raw)
}
