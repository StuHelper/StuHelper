import type { Session } from 'koishi'

import type { GroupConfig } from '../../types'
import type { KeywordModule } from './keyword.module'

const VERIFY_USAGE = '请使用：\n-a 添加关键词\n-r 移除关键词\n--clear 清空关键词\n-l 列出关键词\n-n <true/false> 未匹配关键词自动拒绝\n-w <拒绝词> 设置拒绝时的回复\n多个关键词用英文逗号分隔'

interface VerifyCommandInput {
  host: KeywordModule
  session: Session
  groupConfig: GroupConfig
  value?: unknown
}

export function registerKeywordVerifyCommand(host: KeywordModule): void {
  host.registerCommand({
    name: 'verify',
    desc: '入群验证关键词管理',
    permNode: 'verify',
    permDesc: '管理入群验证关键词',
    usage: '-a 添加关键词，-r 移除，--clear 清空，-l 列出，-n 自动拒绝，-w 设置拒绝词',
  })
    .option('a', '-a <关键词> 添加关键词，多个关键词用英文逗号分隔')
    .option('r', '-r <关键词> 移除关键词，多个关键词用英文逗号分隔')
    .option('clear', '--clear 清除所有关键词')
    .option('l', '-l 列出关键词')
    .option('n', '-n <true/false> 设置未匹配关键词时是否自动拒绝')
    .option('w', '-w <拒绝词> 设置拒绝时的回复')
    .action(async ({ session, options }) => handleVerifyCommand(host, session, options))
}

async function handleVerifyCommand(host: KeywordModule, session: Session, options: any): Promise<string> {
  if (!session.guildId) return '喵呜...这个命令只能在群里用喵...'

  const groupConfig = getVerifyGroupConfig(host, session.guildId)
  const input = { host, session, groupConfig }
  if (options.l) return formatVerifyKeywords(groupConfig)
  if (options.a) return addVerifyKeywords({ ...input, value: options.a })
  if (options.r) return removeVerifyKeywords({ ...input, value: options.r })
  if (options.clear) return clearVerifyKeywords(input)
  if (options.n !== undefined) return setAutoReject({ ...input, value: options.n })
  if (options.w) return setRejectMessage({ ...input, value: options.w })

  return VERIFY_USAGE
}

function getVerifyGroupConfig(host: KeywordModule, guildId: string): GroupConfig {
  const groupConfig = host.data.groupConfig.get(guildId) || {} as GroupConfig
  groupConfig.approvalKeywords = groupConfig.approvalKeywords || []
  if (groupConfig.auto === undefined) groupConfig.auto = 'false'
  if (groupConfig.reject === undefined) groupConfig.reject = '答案错误，请重新申请'
  return groupConfig
}

function formatVerifyKeywords(groupConfig: GroupConfig): string {
  const keywords = groupConfig.approvalKeywords || []
  return `当前群入群审核关键词：\n${keywords.join('、') || '无'}\n自动拒绝状态：${groupConfig.auto}\n拒绝词：${groupConfig.reject}`
}

function addVerifyKeywords(input: VerifyCommandInput): string {
  const { host, session, groupConfig } = input
  const newKeywords = parseKeywordList(String(input.value))
  groupConfig.approvalKeywords.push(...newKeywords)
  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'verify', 'add', `已添加关键词：${newKeywords.join('、')}`)
  return `已经添加了关键词：${newKeywords.join('、')} 喵喵喵~`
}

function removeVerifyKeywords(input: VerifyCommandInput): string {
  const { host, session, groupConfig } = input
  const removed = removeKeywords(groupConfig.approvalKeywords, parseKeywordList(String(input.value)))
  if (removed.length === 0) return '未找到指定的关键词'

  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'verify', 'remove', `已移除关键词：${removed.join('、')}`)
  return `已经把关键词：${removed.join('、')} 删掉啦喵！`
}

function clearVerifyKeywords(input: VerifyCommandInput): string {
  const { host, session, groupConfig } = input
  if (!groupConfig.approvalKeywords.length) return '当前没有任何入群审核关键词喵~'

  groupConfig.approvalKeywords = []
  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'verify', 'clear', '已清除所有关键词')
  return '所有入群审核关键词已清除喵~'
}

function setAutoReject(input: VerifyCommandInput): string {
  const { host, session, groupConfig } = input
  const value = parseExtendedBoolean(input.value)
  if (value === null) return '无效的值，请使用 true/false、1/0、yes/no、y/n 或 on/off'

  groupConfig.auto = value ? 'true' : 'false'
  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'verify', 'auto', `已设置自动拒绝：${groupConfig.auto}`)
  return `自动拒绝状态更新为${groupConfig.auto}`
}

function setRejectMessage(input: VerifyCommandInput): string {
  const { host, session, groupConfig } = input
  const value = String(input.value)
  groupConfig.reject = value
  saveGroupConfig(host, session.guildId, groupConfig)
  void host.log(session, 'verify', 'set', `已设置拒绝词：${value}`)
  return `拒绝词已更新为：${value} 喵喵喵~`
}

function parseKeywordList(input: string): string[] {
  return input.split(',').map((keyword: string) => keyword.trim()).filter((keyword: string) => keyword)
}

function removeKeywords(current: string[], targets: string[]): string[] {
  const removed: string[] = []
  for (const keyword of targets) {
    const index = current.indexOf(keyword)
    if (index > -1) {
      current.splice(index, 1)
      removed.push(keyword)
    }
  }
  return removed
}

function parseExtendedBoolean(input: unknown): boolean | null {
  const value = String(input).toLowerCase()
  if (['true', '1', 'yes', 'y', 'on'].includes(value)) return true
  if (['false', '0', 'no', 'n', 'off'].includes(value)) return false
  return null
}

function saveGroupConfig(host: KeywordModule, guildId: string, groupConfig: GroupConfig): void {
  host.data.groupConfig.set(guildId, groupConfig)
  host.data.groupConfig.flush()
}
