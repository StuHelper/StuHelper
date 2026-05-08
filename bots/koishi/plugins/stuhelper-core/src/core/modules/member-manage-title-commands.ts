import type { Session } from 'koishi'

import type { Config } from '../../types'
import type { MemberManageModule } from './memberManage.module'
import { resolveCommandUserId } from './member-manage-input'

const DEFAULT_MAX_TITLE_BYTES = 18

type TitleConfig = NonNullable<Config['setTitle']>

interface TitleCommandInput {
  readonly host: MemberManageModule
  readonly session: Session
  readonly options: any
  readonly titleConfig: TitleConfig
}

interface SpecialTitleInput {
  readonly host: MemberManageModule
  readonly session: Session
  readonly targetId: string
  readonly value: unknown
  readonly titleConfig: TitleConfig
}

export function registerTitleCommand(host: MemberManageModule): void {
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
    host.logCommand({ session, command: 'title', target: targetId, result: '失败：未知错误', success: false })
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
  host.logCommand({ session, command: 'title', target: targetId, result: `成功：已设置头衔：${title}` })
  return `已经设置好头衔啦喵~`
}

async function removeSpecialTitle(
  host: MemberManageModule,
  session: Session,
  targetId: string,
): Promise<string> {
  await session.bot.internal.setGroupSpecialTitle(session.guildId, targetId, '')
  host.logCommand({ session, command: 'title', target: targetId, result: '成功：已移除头衔' })
  return `已经移除头衔啦喵~`
}
