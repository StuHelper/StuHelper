import type { Session } from 'koishi'

import type { Config, GroupConfig } from '../../types'
import { formatDuration, parseTimeString } from '../../utils'
import type { KeywordModule } from './keyword.module'

const FORBIDDEN_USAGE = '请使用：\n-a 添加关键词\n-r 移除关键词\n--clear 清空关键词\n-l 列出关键词\n-d <true/false> 设置是否自动撤回包含关键词的消息\n-b <true/false> 设置是否启用关键词禁言\n-k <true/false> 设置是否启用关键词踢出\n-t <时长> 设置自动禁言时长\n--echo <true/false> 设置是否启用触发回显\n多个关键词用英文逗号分隔'

interface ForbiddenCommandInput {
  host: KeywordModule
  session: Session
  groupConfig: GroupConfig
  options: any
}

interface ForbiddenFlagInput {
  host: KeywordModule
  session: Session
  groupConfig: GroupConfig
  value: unknown
}

interface ForbiddenFlagDescriptor {
  key: 'autoDelete' | 'autoBan' | 'autoKick'
  command: string
  label: string
}

export function registerKeywordForbiddenCommand(host: KeywordModule): void {
  host.registerCommand({
    name: 'forbidden',
    desc: '禁言关键词管理',
    permNode: 'forbidden',
    permDesc: '管理禁言关键词',
    usage: '-a 添加关键词，-r 移除，--clear 清空，-l 列出，-d/-b/-k 开关，-t 禁言时长',
  })
    .option('a', '-a <关键词> 添加关键词，多个关键词用英文逗号分隔')
    .option('r', '-r <关键词> 移除关键词，多个关键词用英文逗号分隔')
    .option('clear', '--clear 清除所有关键词')
    .option('l', '-l 列出关键词')
    .option('d', '-d <value:string> 设置是否自动撤回包含关键词的消息')
    .option('b', '-b <value:string> 设置是否自动禁言')
    .option('k', '-k <value:string> 设置是否自动踢出')
    .option('t', '-t <时长> 设置自动禁言时长')
    .option('echo', '--echo <value:string> 是否在操作后回显结果')
    .action(async ({ session, options }) => handleForbiddenCommand(host, session, options))
}

async function handleForbiddenCommand(host: KeywordModule, session: Session, options: any): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

  const groupConfig = host.data.groupConfig.get(session.guildId) || {} as GroupConfig
  const input = { host, session, groupConfig, options }
  if (options.l) return formatForbiddenKeywords(input)
  if (options.a) return addForbiddenKeywords(input)
  if (options.r) return removeForbiddenKeywords(input)
  if (options.clear) return clearForbiddenKeywords(input)
  if (options.d !== undefined) {
    return setForbiddenFlag({ ...input, value: options.d }, {
      key: 'autoDelete',
      command: 'recall',
      label: '自动撤回',
    })
  }
  if (options.b !== undefined) {
    return setForbiddenFlag({ ...input, value: options.b }, {
      key: 'autoBan',
      command: 'ban',
      label: '自动禁言',
    })
  }
  if (options.k !== undefined) {
    return setForbiddenFlag({ ...input, value: options.k }, {
      key: 'autoKick',
      command: 'kick',
      label: '自动踢出',
    })
  }
  if (options.t) return setForbiddenDuration({ ...input, value: options.t })
  if (options.echo !== undefined) return setEchoFlag({ ...input, value: options.echo })

  return FORBIDDEN_USAGE
}

function formatForbiddenKeywords(input: ForbiddenCommandInput): string {
  const { host, groupConfig } = input
  const keywords = groupConfig.keywords || []
  const forbiddenConfig = getEffectiveForbiddenConfig(host.config, groupConfig)
  return `全局禁言关键词：\n${host.config.forbidden.keywords.join('、') || '无'}
当前群禁言关键词：\n${keywords.join('、') || '无'}
回显状态：${forbiddenConfig.echo ? '开启' : '关闭'}
自动撤回状态：${forbiddenConfig.autoDelete ? '开启' : '关闭'}
自动禁言状态：${forbiddenConfig.autoBan ? '开启' : '关闭'}
自动踢出状态：${forbiddenConfig.autoKick ? '开启' : '关闭'}
自动禁言时长：${formatDuration(forbiddenConfig.muteDuration)}`
}

function addForbiddenKeywords(input: ForbiddenCommandInput): string {
  const { host, session, groupConfig, options } = input
  const newKeywords = parseKeywordList(options.a)
  groupConfig.keywords = groupConfig.keywords || []
  groupConfig.keywords.push(...newKeywords)
  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'forbidden', 'add', `成功：已添加关键词：${newKeywords.join('、')}`)
  return `已经添加了关键词：${newKeywords.join('、')} 喵喵喵~`
}

function removeForbiddenKeywords(input: ForbiddenCommandInput): string {
  const { host, session, groupConfig, options } = input
  const removed: string[] = []
  if (!groupConfig.keywords) return '当前没有任何禁言关键词喵~'

  for (const keyword of parseKeywordList(options.r)) {
    const index = groupConfig.keywords.indexOf(keyword)
    if (index > -1) {
      groupConfig.keywords.splice(index, 1)
      removed.push(keyword)
    }
  }
  if (removed.length === 0) return '未找到指定的关键词'

  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'forbidden', 'remove', `成功：已移除关键词：${removed.join('、')}`)
  return `已经把关键词：${removed.join('、')} 删掉啦喵！`
}

function clearForbiddenKeywords(input: ForbiddenCommandInput): string {
  const { host, session, groupConfig } = input
  if (!groupConfig.keywords || !groupConfig.keywords.length) return '当前没有任何禁言关键词喵~'

  groupConfig.keywords = []
  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'forbidden', 'clear', '成功：已清除所有关键词')
  return '所有禁言关键词已清除喵~'
}

function setForbiddenFlag(
  input: ForbiddenFlagInput,
  descriptor: ForbiddenFlagDescriptor,
): string {
  const state = parseBooleanOption(input.value)
  if (state === null) return '无效的值，请使用 true/false'

  const forbidden = ensureForbiddenConfig(input.host, input.groupConfig)
  forbidden[descriptor.key] = state
  saveGroupConfig(input.host, input.session.guildId, input.groupConfig)
  void input.host.log(input.session, 'forbidden', descriptor.command, `成功：已设置${descriptor.label}：${state}`)
  return `${descriptor.label}状态更新为${state}`
}

function setForbiddenDuration(input: ForbiddenFlagInput): string {
  const duration = String(input.value)
  try {
    const milliseconds = parseTimeString(duration)
    ensureForbiddenConfig(input.host, input.groupConfig).muteDuration = milliseconds
    saveGroupConfig(input.host, input.session.guildId, input.groupConfig)
    void input.host.log(input.session, 'forbidden', 'set', `成功：已设置禁言时间：${duration}`)
    return `禁言时间已更新为：${duration} 喵喵喵~`
  } catch {
    return `无效的时间格式：${duration}，请使用类似 "1h" 或 "30m" 的格式`
  }
}

function setEchoFlag(input: ForbiddenFlagInput): string {
  const state = parseBooleanOption(input.value)
  if (state === null) return '无效的值，请使用 true/false'

  ensureForbiddenConfig(input.host, input.groupConfig).echo = state
  saveGroupConfig(input.host, input.session.guildId, input.groupConfig)
  void input.host.log(input.session, 'forbidden', 'echo', `成功：已设置回显：${state}`)
  return `回显状态更新为${state}`
}

function getEffectiveForbiddenConfig(config: Config, groupConfig: GroupConfig) {
  return { ...config.forbidden, ...(groupConfig.forbidden || {}) }
}

function ensureForbiddenConfig(host: KeywordModule, groupConfig: GroupConfig) {
  if (!groupConfig.forbidden) {
    groupConfig.forbidden = {
      autoDelete: host.config.forbidden.autoDelete,
      autoBan: host.config.forbidden.autoBan,
      autoKick: host.config.forbidden.autoKick,
      muteDuration: host.config.forbidden.muteDuration,
    }
  }
  return groupConfig.forbidden
}

function parseKeywordList(input: string): string[] {
  return input.split(',').map((keyword: string) => keyword.trim()).filter((keyword: string) => keyword)
}

function parseBooleanOption(input: unknown): boolean | null {
  const value = String(input).toLowerCase()
  if (value === 'true' || value === '1' || value === 'yes' || value === 'y' || value === 'on') return true
  if (value === 'false' || value === '0' || value === 'no' || value === 'n' || value === 'off') return false
  return null
}

function saveGroupConfig(host: KeywordModule, guildId: string, groupConfig: GroupConfig): void {
  host.data.groupConfig.set(guildId, groupConfig)
  host.data.groupConfig.flush()
}
