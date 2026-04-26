import type { Session } from 'koishi'

import type { OrderManageModule } from './orderManage.module'

interface UnbanUsersInput {
  host: OrderManageModule
  session: Session
  currentMutes: ReturnType<typeof getCurrentMutes>
  unbanList: string[]
}

export function registerOrderUnbanCommands(host: OrderManageModule): void {
  registerUnbanRandomCommand(host)
  registerUnbanBatchCommand(host)
}

function registerUnbanRandomCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'unban-random',
    desc: '随机解除若干人禁言',
    args: '<count:number>',
    permNode: 'unban-random',
    permDesc: '随机解除禁言',
    usage: '从当前禁言名单中随机解除指定数量的禁言',
    examples: ['unban-random 3'],
  }).action(async ({ session }, count) => {
    return handleRandomUnbanCommand(host, session, count || 1)
  })
}

async function handleRandomUnbanCommand(
  host: OrderManageModule,
  session: Session,
  count: number,
): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵~'

  const currentMutes = getCurrentMutes(host, session)
  const banList = getActiveMuteUserIds(currentMutes)
  if (banList.length === 0) {
    host.logCommand(session, 'unban-random', session.guildId, '失败：当前没有被禁言的成员', false)
    return '当前没有被禁言的成员喵~'
  }

  const unbanList = host.getRandomElements(banList, count)
  await unbanUsers({ host, session, currentMutes, unbanList })
  host.logCommand(session, 'unban-random', session.guildId, `成功：已随机解除 ${unbanList.length} 人的禁言，解除名单：${unbanList.join(', ')}`)
  return `已随机解除 ${unbanList.length} 人的禁言喵~\n解除名单：\n${unbanList.join(', ')}`
}

function registerUnbanBatchCommand(host: OrderManageModule): void {
  host.registerCommand({
    name: 'unban-batch',
    desc: '批量解除禁言',
    args: '<num:string>',
    permNode: 'unban-batch',
    permDesc: '批量解除禁言',
    usage: '一次性解除多个用户的禁言，按照每个用户已经禁言的百分比来解除',
    examples: ['unban-batch 5'],
  }).action(async ({ session }, num) => handleBatchUnbanCommand(host, session, num))
}

async function handleBatchUnbanCommand(
  host: OrderManageModule,
  session: Session,
  num: string,
): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵~'
  if (!num) return '请提供要解除禁言的用户数量，格式：unban-batch <数量>'

  const count = parseInt(num)
  if (isNaN(count) || count <= 0) return '请提供一个有效的数字，格式：unban-batch <数量>'

  const currentMutes = getCurrentMutes(host, session)
  const banList = getActiveMuteUserIds(currentMutes)
  if (banList.length === 0) {
    host.logCommand(session, 'unban-batch', session.guildId, '失败：当前没有被禁言的成员', false)
    return '当前没有被禁言的成员喵~'
  }

  const unbanList = sortMuteUsersByRemainingRatio(banList, currentMutes).slice(0, count)
  await unbanUsers({ host, session, currentMutes, unbanList })
  host.logCommand(session, 'unban-batch', session.guildId, `成功：已批量解除 ${unbanList.length} 人的禁言，解除名单：${unbanList.join(', ')}`)
  return `已批量解除 ${unbanList.length} 人的禁言喵~\n解除名单：\n${unbanList.join(', ')}`
}

function getCurrentMutes(host: OrderManageModule, session: Session) {
  return host.data.mutes.getAll()[session.guildId] || {}
}

function getActiveMuteUserIds(currentMutes: ReturnType<typeof getCurrentMutes>): string[] {
  const banList: string[] = []
  for (const userId in currentMutes) {
    const muteEndTime = currentMutes[userId].startTime + currentMutes[userId].duration
    if (muteEndTime > Date.now()) {
      banList.push(userId)
    }
  }
  return banList
}

function sortMuteUsersByRemainingRatio(
  banList: string[],
  currentMutes: ReturnType<typeof getCurrentMutes>,
): string[] {
  return banList.sort((a, b) => {
    const aData = currentMutes[a]
    const bData = currentMutes[b]
    const aRemaining = (aData.startTime + aData.duration) - Date.now()
    const bRemaining = (bData.startTime + bData.duration) - Date.now()
    return aRemaining / aData.duration - bRemaining / bData.duration
  })
}

async function unbanUsers(input: UnbanUsersInput): Promise<void> {
  const { host, session, currentMutes, unbanList } = input
  for (const userId of unbanList) {
    await session.bot.muteGuildMember(session.guildId, userId, 0)
    currentMutes[userId].startTime = Date.now()
    currentMutes[userId].duration = 0
  }

  const mutes = host.data.mutes.getAll()
  mutes[session.guildId] = currentMutes
  host.data.mutes.setAll(mutes)
}
