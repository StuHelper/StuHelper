import type { Session } from 'koishi'

import type { AntiRecallModule } from './antirecall.module'
import {
  type AntiRecallConfigOptions,
  formatRecallRecords,
  formatStatusMessage,
  parseConfigUpdates,
  parseRecallQuery,
} from './antirecall-formatters'

interface QueryLogEntry {
  host: AntiRecallModule
  session: Session
  userId: string
  result: string
}

export function registerAntiRecallCommands(host: AntiRecallModule): void {
  registerQueryCommand(host)
  registerConfigCommand(host)
  registerStatusCommand(host)
  registerClearCommand(host)
}

function registerQueryCommand(host: AntiRecallModule): void {
  host.registerCommand({
    name: 'antirecall',
    desc: '查询用户撤回消息记录',
    args: '<input:text>',
    permNode: 'antirecall',
    permDesc: '查询撤回记录',
    usage: '查询指定用户的撤回消息历史',
    examples: ['antirecall @用户', 'antirecall 123456789 5'],
  })
    .alias('撤回查询')
    .usage('查询用户的撤回消息记录\n示例：\nantirecall @用户\nantirecall 123456789\nantirecall @用户 5\nantirecall 123456789 10 群号')
    .example('antirecall @用户')
    .action(async ({ session }, input) => handleRecallQuery(host, session, input))
}

function registerConfigCommand(host: AntiRecallModule): void {
  host.registerCommand({
    name: 'antirecall-config',
    desc: '防撤回功能配置',
    permNode: 'antirecall-config',
    permDesc: '配置防撤回功能',
    usage: '-e 启用/禁用，-d 保留天数，-m 每人最大记录数',
    examples: ['antirecall-config -e true', 'antirecall-config -d 7 -m 100'],
  })
    .alias('防撤回配置')
    .usage('配置群组防撤回功能\n选项：\n  -e <true/false> 启用/禁用\n  -d <days> 设置消息保留天数\n  -m <count> 设置每人最大记录数')
    .option('enabled', '-e <enabled:string> 启用或禁用防撤回功能')
    .option('days', '-d <days:number> 设置保留天数')
    .option('max', '-m <max:number> 设置每用户最大记录数')
    .action(async ({ session, options }) => handleConfigCommand(host, session, options))
}

function registerStatusCommand(host: AntiRecallModule): void {
  host.registerCommand({
    name: 'antirecall.status',
    desc: '查看防撤回功能状态',
    permNode: 'antirecall.status',
    permDesc: '查看防撤回状态',
    usage: '显示当前群防撤回配置和统计信息',
  })
    .action(async ({ session }) => {
      const message = formatStatusMessage(host.getStatus(session.guildId))
      void host.logCommand({
        session,
        command: 'antirecall.status',
        target: session.guildId,
        result: `成功：查询防撤回状态`,
      })
      return message
    })
}

function registerClearCommand(host: AntiRecallModule): void {
  host.registerCommand({
    name: 'antirecall.clear',
    desc: '清理所有撤回记录',
    permNode: 'antirecall.clear',
    permDesc: '清理撤回记录（高危）',
    usage: '清除所有已保存的撤回消息记录',
  })
    .action(async ({ session }) => {
      host.clearAllRecords()
      void host.logCommand({
        session,
        command: 'antirecall.clear',
        target: '',
        result: '成功：清理所有撤回记录',
      })
      return '已清理所有撤回记录'
    })
}

function handleRecallQuery(
  host: AntiRecallModule,
  session: Session,
  input: string,
): string {
  try {
    if (!input) return '请指定要查询的用户\n用法：antirecall @用户 [数量] [群号]'

    const query = parseRecallQuery(input, session.guildId)
    if (!query.targetGuildId) return '请在群聊中使用此命令，或指定群号'
    if (!query.userId) return '无法解析用户ID，请@用户或使用QQ号，并确保格式正确'
    if (!host.isEnabledForGuild(query.targetGuildId)) {
      return `该群组（${query.targetGuildId}）未启用防撤回功能`
    }

    const records = host.getUserRecallRecords(query.targetGuildId, query.userId, query.count)
    if (records.length === 0) {
      logQuerySuccess({ host, session, userId: query.userId, result: `成功：查询到 ${query.targetGuildId} 无记录` })
      return `用户 ${query.userId} 在群 ${query.targetGuildId} 暂无撤回记录`
    }

    const config = host.getAntiRecallConfig(query.targetGuildId)
    logQuerySuccess({
      host,
      session,
      userId: query.userId,
      result: `成功：查询到 ${query.targetGuildId} 撤回记录数 ${records.length}`,
    })
    return formatRecallRecords(records, query.userId, Boolean(config?.showOriginalTime))
  } catch (error) {
    const message = errorMessage(error)
    host.data.writeLog(`[antirecall] 查询撤回记录失败: ${message}`)
    void host.logCommand({ session, command: 'antirecall', target: input, result: `失败: ${message}` })
    return `查询撤回记录失败: ${message}`
  }
}

function handleConfigCommand(host: AntiRecallModule, session: Session, options: AntiRecallConfigOptions): string {
  if (!session.guildId) return '此命令只能在群聊中使用'
  if (Object.keys(options).length === 0) {
    return '请指定要配置的选项：-e (启用/禁用), -d (天数), -m (最大条数)'
  }

  const { updates, messages } = parseConfigUpdates(options)
  if (Object.keys(updates).length === 0) return '未进行任何更改'

  host.updateGuildConfig(session.guildId, updates)
  void host.logCommand({
    session,
    command: 'antirecall-config',
    target: session.guildId,
    result: `更新配置: ${JSON.stringify(updates)}`,
  })
  return `配置已更新：\n${messages.join('\n')}`
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function logQuerySuccess(entry: QueryLogEntry): void {
  void entry.host.logCommand({
    session: entry.session,
    command: 'antirecall',
    target: entry.userId,
    result: entry.result,
  })
}
