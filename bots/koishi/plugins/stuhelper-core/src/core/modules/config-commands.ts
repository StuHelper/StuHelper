import type { Context, Session } from 'koishi'
import type { MemberBlacklistEntry } from '@stuhelper/koishi-shared'

import type { DataManager } from '../data'
import type { Config, WarnRecord } from '../../types'
import type { RuntimeCommand, RuntimeCommandDef } from '../../runtime/types'
import { parseUserId } from '../../utils'
import { showAllConfig } from './config-summary'
import {
  createManualMemberBlacklist,
  listVisibleMemberBlacklists,
  releaseManualMemberBlacklist,
  type MemberBlacklistBackend,
} from './member-blacklist-backend'

type ConfigCommandOptions = {
  readonly t?: boolean
  readonly b?: boolean
  readonly w?: boolean
  readonly a?: string
  readonly r?: string
  readonly global?: boolean
}

export interface ConfigCommandHost {
  readonly ctx: Context
  readonly data: DataManager
  readonly config: Config
  readonly memberBlacklistBackend?: MemberBlacklistBackend
  registerCommand(def: RuntimeCommandDef): RuntimeCommand
  log(
    session: Session,
    command: string,
    target: string,
    result: string,
    success?: boolean,
  ): Promise<void>
}

export function registerConfigCommands(host: ConfigCommandHost): void {
  registerConfigPermissions(host)

  host.registerCommand({
    name: 'config',
    desc: '配置管理',
    permNode: 'config',
    permDesc: '配置管理主命令',
    usage: '-t 显示配置，-b 黑名单管理，-w 警告管理',
  })
    .option('t', '-t 显示所有记录')
    .option('b', '-b 黑名单管理')
    .option('w', '-w 警告管理')
    .option('a', '-a <内容> 添加')
    .option('r', '-r <内容> 移除')
    .option('global', '--global 使用全局黑名单范围')
    .action(async ({ session, options }, content) => {
      if (!session?.guildId) return '喵呜...这个命令只能在群里用喵...'
      return handleConfigCommand(host, session, options, content)
    })
}

function registerConfigPermissions(host: ConfigCommandHost): void {
  const auth = host.ctx.stuhelperGroupCenter.auth
  auth.registerPermission('config.view', '查看配置', '查看所有配置和记录', '配置管理模块')
  auth.registerPermission('config.blacklist', '黑名单管理', '管理黑名单（添加/移除）', '配置管理模块')
  auth.registerPermission('config.warn', '警告管理', '管理警告记录（添加/移除）', '配置管理模块')
}

async function handleConfigCommand(
  host: ConfigCommandHost,
  session: Session,
  options: ConfigCommandOptions,
  content?: string,
): Promise<string> {
  const auth = host.ctx.stuhelperGroupCenter.auth

  if (options.t) {
    if (!auth.check(session, 'config.view')) return '你没有权限查看配置喵...'
    return showAllConfig(host, session)
  }

  if (options.b) {
    if (!auth.check(session, 'config.blacklist')) return '你没有权限管理黑名单喵...'
    return handleBlacklist(host, session, options)
  }

  if (options.w) {
    if (!auth.check(session, 'config.warn')) return '你没有权限管理警告喵...'
    return handleWarns(host, session, options, content)
  }

  return `请使用以下参数：
-t 显示所有配置和记录
-b [-a/-r {QQ号}] 黑名单管理
-w [-a/-r {QQ号} {次数}] 警告管理
使用 verify 命令管理入群审核关键词
使用 forbidden 命令管理禁言关键词
使用 antirepeat 命令管理复读功能`
}

async function handleBlacklist(
  host: ConfigCommandHost,
  session: Session,
  options: ConfigCommandOptions,
): Promise<string> {
  if (options.a) {
    const userId = normalizeBlacklistUserID(options.a)
    await createManualMemberBlacklist(requireMemberBlacklistBackend(host), {
      ...blacklistCommandInput(session, userId, options.global),
    })
    await host.log(session, 'config -b -a', userId, '添加成功')
    await host.ctx.stuhelperGroupCenter.pushMessage(
      session.bot,
      `[黑名单] 用户 ${userId} 被加入${formatScopeLabel(options.global)}黑名单`,
      'blacklist',
    )
    return `已将 ${userId} 加入${formatScopeLabel(options.global)}黑名单喵~`
  }

  if (options.r) {
    const userId = normalizeBlacklistUserID(options.r)
    await releaseManualMemberBlacklist(requireMemberBlacklistBackend(host), {
      ...blacklistCommandInput(session, userId, options.global),
    })
    await host.log(session, 'config -b -r', userId, '移除成功')
    await host.ctx.stuhelperGroupCenter.pushMessage(
      session.bot,
      `[黑名单] 用户 ${userId} 被移出${formatScopeLabel(options.global)}黑名单`,
      'blacklist',
    )
    return `已将 ${userId} 从${formatScopeLabel(options.global)}黑名单移除啦！`
  }

  const entries = await listVisibleMemberBlacklists(
    requireMemberBlacklistBackend(host),
    session.platform,
    session.guildId!,
  )
  return `=== 当前黑名单 ===\n${formatBlacklist(entries) || '无记录'}`
}

async function handleWarns(
  host: ConfigCommandHost,
  session: Session,
  options: ConfigCommandOptions,
  content?: string,
): Promise<string> {
  const guildWarns = host.data.warns.get(session.guildId!) || {}
  const countDelta = parseInt(content ?? '') || 1

  if (options.a) {
    guildWarns[options.a] = guildWarns[options.a] || { count: 0, timestamp: 0 }
    guildWarns[options.a].count += countDelta
    guildWarns[options.a].timestamp = Date.now()
    host.data.warns.set(session.guildId!, guildWarns)
    host.data.warns.flush()
    void host.log(session, 'config -w -a', options.a, `增加到 ${guildWarns[options.a].count} 次`)
    return `已增加 ${options.a} 的警告次数，当前为：${guildWarns[options.a].count}`
  }

  if (options.r) {
    return handleWarnRemoval(host, session, options.r, countDelta)
  }

  return `=== 当前群警告记录 ===\n${formatWarns(guildWarns) || '无记录'}`
}

function handleWarnRemoval(
  host: ConfigCommandHost,
  session: Session,
  userId: string,
  countDelta: number,
): string {
  const guildWarns = host.data.warns.get(session.guildId!) || {}
  if (!guildWarns[userId]) return '未找到该用户的警告记录'

  guildWarns[userId].count -= countDelta
  const resultMsg = formatWarnRemovalResult(host, session, userId, guildWarns)

  if (Object.keys(guildWarns).length === 0) {
    host.data.warns.delete(session.guildId!)
  } else {
    host.data.warns.set(session.guildId!, guildWarns)
  }
  host.data.warns.flush()
  return resultMsg
}

function formatWarnRemovalResult(
  host: ConfigCommandHost,
  session: Session,
  userId: string,
  guildWarns: WarnRecord,
): string {
  if (guildWarns[userId].count <= 0) {
    delete guildWarns[userId]
    void host.log(session, 'config -w -r', userId, '记录已移除')
    return `已移除 ${userId} 的警告记录`
  }

  guildWarns[userId].timestamp = Date.now()
  void host.log(session, 'config -w -r', userId, `减少到 ${guildWarns[userId].count} 次`)
  return `已减少 ${userId} 的警告次数，当前为：${guildWarns[userId].count}`
}

function requireMemberBlacklistBackend(host: ConfigCommandHost): MemberBlacklistBackend {
  if (!host.memberBlacklistBackend) {
    throw new Error('member blacklist backend client is required for config blacklist command')
  }
  return host.memberBlacklistBackend
}

function blacklistCommandInput(session: Session, subjectID: string, global?: boolean) {
  return {
    platform: session.platform,
    guildID: session.guildId!,
    subjectID,
    operatorQQID: session.userId,
    rawCommand: session.content?.trim() || '',
    global,
  }
}

function normalizeBlacklistUserID(value: string): string {
  return parseUserId(value)
}

function formatScopeLabel(global?: boolean): string {
  return global ? '全局' : '本群'
}

function formatBlacklist(entries: readonly MemberBlacklistEntry[]): string {
  return entries
    .map((entry) => `用户 ${entry.subjectID}：${formatBlacklistScope(entry)} / ${formatShanghaiTime(Date.parse(entry.createdAt))}`)
    .join('\n')
}

function formatBlacklistScope(entry: MemberBlacklistEntry): string {
  return entry.scopeType === 'global' ? '全局' : `群 ${entry.guildID}`
}

function formatWarns(guildWarns: WarnRecord): string {
  return Object.entries(guildWarns)
    .filter(([, data]) => data.count > 0)
    .map(([userId, data]) => `用户 ${userId}：${data.count} 次 (${formatShanghaiTime(data.timestamp)})`)
    .filter(Boolean)
    .join('\n')
}

function formatShanghaiTime(timestamp: number): string {
  return new Date(timestamp).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
}
