import type { Session } from 'koishi'
import { createPlatformClient } from '@stuhelper/koishi-shared'

import type { MemberManageModule } from './memberManage.module'
import {
  parseKickInput,
  resolveCommandUserId,
  type KickInput,
  type KickOptions,
} from './member-manage-input'
import { registerTitleCommand } from './member-manage-title-commands'

interface AdminCommandInput { host: MemberManageModule; session: Session; user: unknown; enabled: boolean }
interface UnbanMemberInput { host: MemberManageModule; session: Session; userId: string; now: number }

export { parseKickInput }

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
    .action(async ({ session, options }, input) => handleKickCommand({ host, session, input, options }))
}

async function handleKickCommand(commandInput: {
  readonly host: MemberManageModule
  readonly session: Session
  readonly input: string
  readonly options?: KickOptions
}): Promise<string> {
  const { host, session, input, options = {} } = commandInput
  if (!input?.trim()) {
    host.logCommand({ session, command: 'kick', target: 'none', result: '失败：缺少必要参数', success: false })
    return '喵呜...请输入正确的用户（@或QQ号）'
  }

  const kickInput = parseKickInput(input, session.guildId, options)
  if (!kickInput.userId) {
    host.logCommand({ session, command: 'kick', target: 'none', result: '失败：无法读取目标用户', success: false })
    return '喵呜...请输入正确的用户（@或QQ号）'
  }
  if (!kickInput.targetGroup) {
    host.logCommand({ session, command: 'kick', target: kickInput.userId, result: '失败：缺少群号', success: false })
    return '喵呜...请在群聊中执行，或显式传入群号'
  }

  try {
    await session.bot.kickGuildMember(kickInput.targetGroup, kickInput.userId, kickInput.black)
    if (kickInput.black) {
      const result = await handleBlackKick(host, session, kickInput)
      if (result) return result
      return `已把坏人 ${kickInput.userId} 踢出去并加入黑名单啦喵！`
    }

    host.logCommand({ session, command: 'kick', target: kickInput.userId, result: `成功：移出群聊 ${kickInput.targetGroup}` })
    return `已把 ${kickInput.userId} 踢出去喵~`
  } catch (error) {
    host.logCommand({ session, command: 'kick', target: kickInput.userId, result: '失败：未知错误', success: false })
    return `喵呜...踢出失败了：${error.message}`
  }
}

async function handleBlackKick(
  host: MemberManageModule,
  session: Session,
  input: KickInput,
): Promise<string | null> {
  if (!input.userId) throw new Error('kick blacklist requires userId')
  try {
    await createPlatformClient(host.platformConfig).createMemberBlacklist({
      platform: session.platform,
      subjectType: 'qq_user',
      subjectID: input.userId,
      scopeType: 'guild',
      guildID: input.targetGroup,
      source: 'kick_blacklist',
      reasonCode: 'manual_kick_blacklist',
      reasonText: 'QQ command kick blacklist',
      createdFrom: 'qq_command',
      operatorID: session.userId,
      metadata: { rawCommand: session.content || '', targetGuildID: input.targetGroup },
    })
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    host.logCommand({
      session,
      command: 'kick',
      target: input.userId,
      result: `部分成功：已踢出但加入黑名单失败：${message}`,
      success: false,
    })
    return `已把 ${input.userId} 踢出群 ${input.targetGroup}，但加入黑名单失败：${message}`
  }
  host.logCommand({ session, command: 'kick', target: input.userId, result: `成功：移出群聊并加入黑名单：${input.targetGroup}` })
  await host.ctx.stuhelperGroupCenter.pushMessage(
    session.bot,
    `[黑名单] 用户 ${input.userId} 被踢出群 ${input.targetGroup} 并加入黑名单`,
    'blacklist',
  )
  return null
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
    host.logCommand({ session, command: commandName, target: userId, result: enabled ? '成功：已设置为管理员' : '成功：已取消管理员' })
    return enabled ? `已将 ${userId} 设置为管理员喵~` : `已取消 ${userId} 的管理员权限喵~`
  } catch (error) {
    host.logCommand({ session, command: commandName, target: userId, result: '失败：未知错误', success: enabled ? false : undefined })
    return enabled ? `设置失败了喵...${error.message}` : `取消失败了喵...${error.message}`
  }
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
    host.logCommand({ session, command: 'unban-allppl', target: session.guildId, result: `成功：已解除 ${count} 人的禁言` })
    return count > 0 ? `已解除 ${count} 人的禁言啦！` : '当前没有被禁言的成员喵~'
  } catch (error) {
    host.logCommand({ session, command: 'unban-allppl', target: session.guildId, result: '失败：未知错误', success: false })
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
    host.ctx.logger('stuhelper-core:member-manage').error('解除用户 %s 禁言失败: %o', userId, error)
    return 0
  }
}
