import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Session } from 'koishi'

import type { Config } from '../../types'
import type { ReportModule } from './report.module'
import { parseViolationInfo, registerReportCommands } from './report-commands.ts'
import { ViolationLevel } from './report-types.ts'

type CommandAction = (argv: {
  readonly session?: Session
  readonly options: Record<string, unknown>
}) => string | Promise<string>

interface CapturedCommandChain {
  option(...args: unknown[]): CapturedCommandChain
  action(fn: CommandAction): CapturedCommandChain
}

interface ReportHostOverrides {
  readonly callModeration?: ReportModule['callModeration']
  readonly handleViolation?: ReportModule['handleViolation']
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('report command source uses explicit command boundary types', () => {
  const source = readFileSync(resolve(modulesDir, './report-commands.ts'), 'utf8')

  assert.doesNotMatch(source, /session: any/)
  assert.doesNotMatch(source, /options: any/)
  assert.doesNotMatch(source, /message: any/)
  assert.doesNotMatch(source, /catch \(error: any\)/)
  assert.doesNotMatch(source, /error: any/)
  assert.doesNotMatch(source, /quote\(session: any\)/)
  assert.match(source, /interface ReportCommandOptions/)
  assert.match(source, /interface ReportedMessage/)
})

test('parseViolationInfo rejects moderation JSON with an out-of-range violation level', () => {
  assert.throws(() => parseViolationInfo(JSON.stringify({
    level: 99,
    reason: 'invalid',
    action: [],
  })), /level/)
})

test('parseViolationInfo rejects actions missing required numeric fields', () => {
  assert.throws(() => parseViolationInfo(JSON.stringify({
    level: ViolationLevel.MEDIUM,
    reason: 'missing ban time',
    action: [{ type: 'ban' }],
  })), /ban time/)
})

test('parseViolationInfo accepts a bounded moderation action payload', () => {
  const result = parseViolationInfo(JSON.stringify({
    level: ViolationLevel.MEDIUM,
    reason: 'validated',
    action: [{ type: 'ban', time: 600 }, { type: 'warn', count: 1 }],
    reporterPenalty: { shouldLimit: true, duration: 30, reason: 'abuse' },
  }))

  assert.equal(result.level, ViolationLevel.MEDIUM)
  assert.deepEqual(result.action.map((item) => item.type), ['ban', 'warn'])
})

test('report command reports non-Error moderation failures without throwing again', async () => {
  const { actions, host, logs } = createReportCommandHost({
    callModeration: async () => {
      throw 'moderation offline'
    },
  })
  registerReportCommands(host)

  const result = await runReportCommand(actions, createReportSession(), {})

  assert.match(result, /举报处理失败：moderation offline/)
  assert.ok(host.reportBans['reporter:guild-1'])
  assert.equal(logs.at(-1)?.command, 'report-banned')
  assert.match(logs.at(-1)?.details ?? '', /举报处理失败\(moderation offline\)，已限制使用/)
})

test('report command uses the default reporter penalty duration in output and logs', async () => {
  const { actions, host, logs } = createReportCommandHost({
    callModeration: async () => JSON.stringify({
      level: ViolationLevel.NONE,
      reason: 'false report',
      action: [],
      reporterPenalty: {
        shouldLimit: true,
        reason: 'abuse',
      },
    }),
    handleViolation: async () => '该消息未被判定为违规内容。',
  })
  registerReportCommands(host)

  const result = await runReportCommand(actions, createReportSession(), {})

  assert.match(result, /已被暂时限制举报功能60分钟/)
  assert.ok(host.reportBans['reporter:guild-1'])
  assert.equal(logs.at(-1)?.command, 'report-banned')
  assert.match(logs.at(-1)?.details ?? '', /AI判定: abuse，限制60分钟/)
})

function createReportCommandHost(overrides: ReportHostOverrides = {}): {
  readonly actions: Map<string, CommandAction>
  readonly host: ReportModule
  readonly logs: Array<{ command: string; target: string; details: string }>
} {
  const actions = new Map<string, CommandAction>()
  const logs: Array<{ command: string; target: string; details: string }> = []
  const reportBans: ReportModule['reportBans'] = {}
  const reportedMessages: ReportModule['reportedMessages'] = {}
  const guildMessages: ReportModule['guildMessages'] = {}

  const host = {
    config: {
      report: {
        enabled: true,
        autoProcess: true,
      },
    } as Config,
    ctx: {},
    reportBans,
    reportedMessages,
    guildMessages,
    getMinUnlimitedAuthority: () => 2,
    getReportCooldownDuration: () => 60 * 60 * 1000,
    getMaxReportTime: () => 30,
    getDefaultPrompt: () => 'review {content}',
    getContextPrompt: () => '{context}\n{content}',
    getReportGuildConfig: () => ({ enabled: true, autoProcess: true, includeContext: false }),
    getViolationLevelText: () => '未',
    callModeration: overrides.callModeration ?? (async () => JSON.stringify({
      level: ViolationLevel.NONE,
      reason: 'ok',
      action: [],
    })),
    handleViolation: overrides.handleViolation ?? (async () => '该消息未被判定为违规内容。'),
    logCommand: async (entry: { readonly command: string; readonly target: string; readonly details: string }) => {
      logs.push(entry)
    },
    registerCommand(def: { readonly name: string }) {
      assert.equal(def.name, 'report')
      const chain: CapturedCommandChain = {
        option: () => chain,
        action(fn: CommandAction) {
          actions.set(def.name, fn)
          return chain
        },
      }
      return chain as unknown as ReturnType<ReportModule['registerCommand']>
    },
  }

  Object.assign(host.ctx, {
    database: undefined,
  })

  return {
    actions,
    host: host as unknown as ReportModule,
    logs,
  }
}

function createReportSession(): Session {
  return {
    platform: 'qq',
    guildId: 'guild-1',
    channelId: 'channel-1',
    userId: 'reporter',
    selfId: 'bot-self',
    messageId: 'report-message',
    quote: { id: 'reported-message' },
    bot: {
      getMessage: async () => ({
        content: 'reported content',
        userId: 'target-user',
        timestamp: Date.now(),
      }),
    },
  } as unknown as Session
}

async function runReportCommand(
  actions: Map<string, CommandAction>,
  session: Session,
  options: Record<string, unknown>,
): Promise<string> {
  const action = actions.get('report')
  if (!action) throw new Error('report action was not registered')
  return action({ session, options })
}
