import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Session } from 'koishi'

import type { Config, ReportConfig } from '../../types'
import type { ReportModule } from './report.module'
import { registerReportConfigCommand } from './report-config-command'

type CommandAction = (argv: {
  readonly session?: Session
  readonly options: Record<string, unknown>
}) => string | Promise<string>

interface CapturedCommandChain {
  option(...args: unknown[]): CapturedCommandChain
  action(fn: CommandAction): CapturedCommandChain
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('report config command source uses explicit option and session types', () => {
  const source = readModuleFile('./report-config-command.ts')

  assert.doesNotMatch(source, /session: any/)
  assert.doesNotMatch(source, /options: any/)
  assert.doesNotMatch(source, /hasGuildConfigChanges\(options: any\)/)
  assert.match(source, /interface ReportConfigOptions/)
  assert.match(source, /readonly session: Session/)
})

test('report-config updates typed guild report options', async () => {
  const { actions, updates, host } = createReportHost()
  registerReportConfigCommand(host)

  const result = await runCommand(actions, createSession('guild-1'), {
    enabled: false,
    auto: false,
    context: true,
    'context-size': 10,
  })

  assert.match(result, /^举报功能配置已更新/)
  assert.equal(updates.length, 1)
  assert.deepEqual(updates[0].guildConfigs?.['guild-1'], {
    enabled: false,
    includeContext: true,
    contextSize: 10,
    autoProcess: false,
  })
})

test('report-config lists defaults without persisting when no options change', async () => {
  const { actions, updates, host } = createReportHost()
  registerReportConfigCommand(host)

  const result = await runCommand(actions, createSession('guild-1'), {})

  assert.match(result, /群 guild-1 的举报功能配置：/)
  assert.match(result, /状态: 已启用/)
  assert.match(result, /上下文消息数量: 5/)
  assert.equal(updates.length, 0)
})

test('report-config rejects out-of-range context size before saving', async () => {
  const { actions, updates, host } = createReportHost()
  registerReportConfigCommand(host)

  assert.equal(
    await runCommand(actions, createSession('guild-1'), { 'context-size': 21 }),
    '上下文消息数量必须在1-20之间',
  )
  assert.equal(updates.length, 0)
})

function readModuleFile(relativePath: string): string {
  return readFileSync(resolve(modulesDir, relativePath), 'utf8')
}

function createReportHost(): {
  readonly actions: Map<string, CommandAction>
  readonly updates: ReportConfig[]
  readonly host: ReportModule
} {
  const actions = new Map<string, CommandAction>()
  const updates: ReportConfig[] = []
  const reportConfig: ReportConfig = {
    enabled: true,
    authority: 1,
    autoProcess: true,
    maxReportCooldown: 60,
    minAuthorityNoLimit: 2,
    maxReportTime: 30,
    guildConfigs: {},
  }

  const host = {
    config: {
      report: reportConfig,
    } as Config,
    ctx: {
      stuhelperGroupCenter: {
        settings: {
          update: async (value: { readonly report: ReportConfig }) => {
            updates.push(value.report)
          },
        },
      },
    },
    logCommand: async () => undefined,
    registerCommand(def: { readonly name: string }) {
      assert.equal(def.name, 'report-config')
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

  return {
    actions,
    updates,
    host: host as unknown as ReportModule,
  }
}

function createSession(guildId: string): Session {
  return {
    guildId,
    userId: 'operator',
    platform: 'qq',
  } as Session
}

async function runCommand(
  actions: Map<string, CommandAction>,
  session: Session,
  options: Record<string, unknown>,
): Promise<string> {
  const action = actions.get('report-config')
  if (!action) throw new Error('report-config action was not registered')
  return action({ session, options })
}
