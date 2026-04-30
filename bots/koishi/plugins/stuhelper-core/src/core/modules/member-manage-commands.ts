import type { Session } from 'koishi'

import { parseUserId } from '../../utils'
import type { Config } from '../../types'
import type { MemberManageModule } from './memberManage.module'

const DEFAULT_MAX_TITLE_BYTES = 18

interface KickInput {
  userId: string | null
  targetGroup?: string
  black: boolean
}

type TitleConfig = NonNullable<Config['setTitle']>
interface AdminCommandInput { host: MemberManageModule; session: Session; user: unknown; enabled: boolean }
interface TitleCommandInput { host: MemberManageModule; session: Session; options: any; titleConfig: TitleConfig }
interface SpecialTitleInput { host: MemberManageModule; session: Session; targetId: string; value: unknown; titleConfig: TitleConfig }
interface UnbanMemberInput { host: MemberManageModule; session: Session; userId: string; now: number }

export function registerMemberManageCommands(host: MemberManageModule): void {
  registerKickCommand(host)
  registerAdminCommands(host)
  registerUnbanAllPplCommand(host)
  registerTitleCommand(host)
}

function registerKickCommand(host: MemberManageModule): void {
  host.registerCommand({
    name: 'kick',
    desc: '踢出用户',
    args: '<input:text>',
    permNode: 'kick',
    permDesc: '踢出群成员',
    usage: '支持 @用户 或 QQ号，可指定群号，-b 加入黑名单',
    examples: ['kick @用户', 'kick 123456789', 'kick @用户 -b'],
  })
    .example('kick @用户')
    .example('kick 123456789')
    .example('kick @用户 群号')
    .example('kick @用户 -b')
    .example('kick 123456789 -b 群号')
    .option('black', '-b 加入黑名单')
    .action(async ({ session }, input) => handleKickCommand(host, session, input))
}

async function handleKickCommand(host: MemberManageModule, session: Session, input: string): Promise<string> {
  if (!input?.trim()) {
    host.logCommand(session, 'kick', 'none', '失败：缺少必要参数', false)
    return '喵呜...请输入正确的用户（@或QQ号）'
  }

  const kickInput = parseKickInput(input, session.guildId)
  if (!kickInput.userId) {
    host.logCommand(session, 'kick', 'none', '失败：无法读取目标用户', false)
    return '喵呜...请输入正确的用户（@或QQ号）'
  }
  if (!kickInput.targetGroup) {
    host.logCommand(session, 'kick', kickInput.userId, '失败：缺少群号', false)
    return '喵呜...请在群聊中执行，或显式传入群号'
  }

  try {
    await session.bot.kickGuildMember(kickInput.targetGroup, kickInput.userId, kickInput.black)
    if (kickInput.black) {
      await handleBlackKick(host, session, kickInput)
      return `已把坏人 ${kickInput.userId} 踢出去并加入黑名单啦喵！`
    }

    host.logCommand(session, 'kick', kickInput.userId, `成功：移出群聊 ${kickInput.targetGroup}`)
    return `已把 ${kickInput.userId} 踢出去喵~`
  } catch (error) {
    host.logCommand(session, 'kick', kickInput.userId, `失败：未知错误`, false)
    return `喵呜...踢出失败了：${error.message}`
  }
}

function parseKickInput(input: string, defaultGuildId?: string): KickInput {
  const black = input.includes('-b')
  const normalized = input.replace(/-b/g, '').replace(/\s+/g, ' ').trim()
  const [target, groupId] = splitKickArgs(normalized)

  return {
    userId: resolveTargetUserId(target),
    targetGroup: groupId || defaultGuildId,
    black,
  }
}

function splitKickArgs(input: string): string[] {
  const source = input.trim()
  if (!source.includes('<at')) return source.split(/\s+/).filter(Boolean)

  const atMatch = source.match(/<at[^>]+>/)
  if (!atMatch) return source.split(/\s+/).filter(Boolean)

  const atPart = atMatch[0]
  const restPart = source.replace(atPart, '').trim()
  return [atPart, ...restPart.split(/\s+/)].filter(Boolean)
}

function resolveTargetUserId(target: string): string | null {
  try {
    if (target?.startsWith('<at')) {
      const match = target.match(/id="(\d+)"/)
      if (match) return match[1]
    } else {
      return parseUserId(target)
    }
  } catch {
    return parseUserId(target)
  }
  return null
}

function resolveCommandUserId(user: unknown): string {
  const raw = String(user || '').trim()
  if (!raw) return ''
  const [, platformUserId] = raw.split(':')
  return platformUserId || resolveTargetUserId(raw) || ''
}

async function handleBlackKick(
  host: MemberManageModule,
  session: Session,
  input: KickInput,
): Promise<void> {
  const blacklist = host.data.blacklist.getAll()
  blacklist[input.userId] = { userId: input.userId, timestamp: Date.now() }
  host.data.blacklist.setAll(blacklist)
  host.logCommand(session, 'kick', input.userId, `成功：移出群聊并加入黑名单：${input.targetGroup}`)
  await host.ctx.stuhelperGroupCenter.pushMessage(
    session.bot,
    `[黑名单] 用户 ${input.userId} 被踢出群 ${input.targetGroup} 并加入黑名单`,
    'blacklist',
  )
}

function registerAdminCommands(host: MemberManageModule): void {
  registerAdminCommand(host, true)
  registerAdminCommand(host, false)
}

function registerAdminCommand(host: MemberManageModule, enabled: boolean): void {
  const commandName = enabled ? 'admin' : 'unadmin'
  host.registerCommand({
    name: commandName,
    desc: enabled ? '设置管理员' : '取消管理员',
    args: '<user:user>',
    permNode: commandName,
    permDesc: enabled ? '设置群管理员' : '取消群管理员',
    examples: [`${commandName} @用户`],
  })
    .example(`${commandName} @用户`)
    .action(async ({ session }, user) => handleAdminCommand({ host, session, user, enabled }))
}

async function handleAdminCommand(input: AdminCommandInput): Promise<string> {
  const { host, session, user, enabled } = input
  if (!user) return '请指定用户'
  if (!session.guildId) return '请在群聊中执行该命令。'

  const commandName = enabled ? 'admin' : 'unadmin'
  const userId = resolveCommandUserId(user)
  if (!userId) return '请指定正确的用户'

  try {
    const internal = session.bot.internal
    if (typeof internal?.setGroupAdmin !== 'function') {
      throw new Error('当前适配器不支持 OneBot set_group_admin')
    }
    await internal.setGroupAdmin(session.guildId, userId, enabled)
    host.logCommand(session, commandName, userId, enabled ? '成功：已设置为管理员' : '成功：已取消管理员')
    return enabled ? `已将 ${userId} 设置为管理员喵~` : `已取消 ${userId} 的管理员权限喵~`
  } catch (error) {
    host.logCommand(session, commandName, userId, `失败：未知错误`, enabled ? false : undefined)
    return enabled ? `设置失败了喵...${error.message}` : `取消失败了喵...${error.message}`
  }
}

function registerTitleCommand(host: MemberManageModule): void {
  const titleConfig = host.config.setTitle || { enabled: false, authority: 3, maxLength: DEFAULT_MAX_TITLE_BYTES }

  host.registerCommand({
    name: 'title',
    desc: '群头衔管理',
    permNode: 'title',
    permDesc: '设置群头衔',
    usage: '-s <文本> 设置头衔，-r 移除头衔，-u @用户 指定用户',
    examples: ['title -s 大佬', 'title -r', 'title -s 萌新 -u @用户'],
  })
    .option('s', '-s <text> 设置头衔')
    .option('r', '-r 移除头衔')
    .option('u', '-u <user:user> 指定用户')
    .action(async ({ session, options }) => handleTitleCommand({ host, session, options, titleConfig }))
}

async function handleTitleCommand(input: TitleCommandInput): Promise<string> {
  const { host, session, options, titleConfig } = input
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'
  if (!titleConfig.enabled) return '喵呜...头衔功能未启用...'

  const targetId = options.u ? resolveCommandUserId(options.u) : session.userId
  if (!targetId) return '请指定正确的用户'

  try {
    if (options.s) return await setSpecialTitle({ host, session, targetId, value: options.s, titleConfig })
    if (options.r) return await removeSpecialTitle(host, session, targetId)
    return '请使用 -s <文本> 设置头衔或 -r 移除头衔\n可选 -u @用户 为指定用户设置'
  } catch (error) {
    host.logCommand(session, 'title', targetId, `失败：未知错误`, false)
    return `出错啦喵...${error.message}`
  }
}

async function setSpecialTitle(input: SpecialTitleInput): Promise<string> {
  const { host, session, targetId, value, titleConfig } = input
  const title = value.toString()
  const maxLength = titleConfig.maxLength || DEFAULT_MAX_TITLE_BYTES
  if (new TextEncoder().encode(title).length > maxLength) {
    return `喵呜...头衔太长啦！最多只能有 ${maxLength} 个字节哦~`
  }
  await session.bot.internal.setGroupSpecialTitle(session.guildId, targetId, title)
  host.logCommand(session, 'title', targetId, `成功：已设置头衔：${title}`)
  return `已经设置好头衔啦喵~`
}

async function removeSpecialTitle(
  host: MemberManageModule,
  session: Session,
  targetId: string,
): Promise<string> {
  await session.bot.internal.setGroupSpecialTitle(session.guildId, targetId, '')
  host.logCommand(session, 'title', targetId, `成功：已移除头衔`)
  return `已经移除头衔啦喵~`
}

function registerUnbanAllPplCommand(host: MemberManageModule): void {
  host.registerCommand({
    name: 'unban-allppl',
    desc: '解除所有人禁言',
    permNode: 'unban-allppl',
    permDesc: '批量解除所有禁言',
    usage: '解除当前群所有被禁言成员的禁言状态',
  }).action(async ({ session }) => handleUnbanAllPplCommand(host, session))
}

async function handleUnbanAllPplCommand(host: MemberManageModule, session: Session): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵~'

  try {
    const count = await unbanCurrentGuildMembers(host, session)
    host.logCommand(session, 'unban-allppl', session.guildId, `成功：已解除 ${count} 人的禁言`)
    return count > 0 ? `已解除 ${count} 人的禁言啦！` : '当前没有被禁言的成员喵~'
  } catch (error) {
    host.logCommand(session, 'unban-allppl', session.guildId, `失败：未知错误`, false)
    return `出错啦喵...${error}`
  }
}

async function unbanCurrentGuildMembers(host: MemberManageModule, session: Session): Promise<number> {
  const mutes = host.data.mutes.getAll()
  const currentMutes = mutes[session.guildId] || {}
  const now = Date.now()
  let count = 0

  for (const userId in currentMutes) {
    if (currentMutes[userId].leftGroup) continue
    count += await unbanMemberIfMuted({ host, session, userId, now })
  }

  mutes[session.guildId] = currentMutes
  host.data.mutes.setAll(mutes)
  return count
}

async function unbanMemberIfMuted(input: UnbanMemberInput): Promise<number> {
  const { host, session, userId, now } = input
  try {
    const currentMutes = host.data.mutes.getAll()[session.guildId] || {}
    const muteEndTime = currentMutes[userId].startTime + currentMutes[userId].duration
    if (now >= muteEndTime) {
      delete currentMutes[userId]
      return 0
    }

    const memberInfo = await session.bot.internal.getGroupMemberInfo(session.guildId, userId, false)
    delete currentMutes[userId]
    if (memberInfo.shut_up_timestamp <= 0) return 0

    await session.bot.muteGuildMember(session.guildId, userId, 0)
    return 1
  } catch (error) {
    console.error(`解除用户 ${userId} 禁言失败:`, error)
    return 0
  }
}
