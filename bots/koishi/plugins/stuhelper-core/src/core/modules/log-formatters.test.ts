import test from 'node:test'
import assert from 'node:assert/strict'

import type { CommandLogRecord } from './log.module'
import { filterLogs, formatStats } from './log-formatters'

const records: CommandLogRecord[] = [
  createLog({
    command: 'warn.add',
    userId: '1001',
    username: 'Alice',
    guildId: '2001',
    success: true,
    timestamp: '2026-05-27T01:00:00.000Z',
  }),
  createLog({
    command: 'ban.kick',
    userId: '1002',
    username: 'Bob',
    guildId: '2002',
    success: false,
    error: 'permission denied',
    timestamp: '2026-05-27T02:00:00.000Z',
  }),
  createLog({
    command: 'warn.clear',
    userId: '1001',
    username: 'Alice',
    guildId: '2001',
    success: true,
    timestamp: '2026-05-27T03:00:00.000Z',
  }),
]

test('filterLogs applies typed command, guild, user, failure and time filters', () => {
  assert.deepEqual(filterLogs(records, { command: 'warn', guild: '2001' }).map((log) => log.command), [
    'warn.add',
    'warn.clear',
  ])
  assert.deepEqual(filterLogs(records, { user: '1002', failed: true }).map((log) => log.command), ['ban.kick'])
  assert.deepEqual(filterLogs(records, { since: '2026-05-27T02:30:00.000Z' }).map((log) => log.command), ['warn.clear'])
})

test('formatStats renders command usage totals from typed stats options', () => {
  const message = formatStats(records, { limit: 2 })

  assert.match(message, /命令使用统计/)
  assert.match(message, /总记录: 3 条/)
  assert.match(message, /warn\.add/)
  assert.match(message, /ban\.kick/)
})

function createLog(input: Partial<CommandLogRecord> & Pick<CommandLogRecord, 'command' | 'timestamp'>): CommandLogRecord {
  return {
    id: input.id || input.command,
    timestamp: input.timestamp,
    userId: input.userId || 'user',
    username: input.username,
    userAuthority: input.userAuthority,
    guildId: input.guildId,
    guildName: input.guildName,
    channelId: input.channelId,
    platform: input.platform || 'onebot',
    command: input.command,
    args: input.args || [],
    options: input.options || {},
    success: input.success ?? true,
    error: input.error,
    executionTime: input.executionTime ?? 1,
    result: input.result,
    messageId: input.messageId,
    isPrivate: input.isPrivate ?? false,
  }
}
