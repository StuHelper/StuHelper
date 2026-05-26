import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Session } from 'koishi'

import type { Config, GroupConfig } from '../../types'
import type { KeywordModule } from './keyword.module'
import { registerKeywordForbiddenCommand } from './keyword-forbidden-command'
import { registerKeywordVerifyCommand } from './keyword-verify-command'

type CommandAction = (argv: {
  readonly session: Session
  readonly options: Record<string, unknown>
}) => string | Promise<string>

interface CapturedCommandChain {
  option(...args: unknown[]): CapturedCommandChain
  action(fn: CommandAction): CapturedCommandChain
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('keyword command source uses explicit option and forbidden config types', () => {
  const forbiddenSource = readModuleFile('./keyword-forbidden-command.ts')
  const verifySource = readModuleFile('./keyword-verify-command.ts')
  const middlewareSource = readModuleFile('./keyword-middleware.ts')

  for (const source of [forbiddenSource, verifySource, middlewareSource]) {
    assert.doesNotMatch(source, /options: any/)
    assert.doesNotMatch(source, /forbiddenConfig: any/)
  }

  assert.match(forbiddenSource, /interface ForbiddenCommandOptions/)
  assert.match(verifySource, /interface VerifyCommandOptions/)
  assert.match(middlewareSource, /type EffectiveForbiddenConfig/)
})

test('forbidden command updates typed keyword options and forbidden settings', async () => {
  const { actions, groupConfigs, host } = createKeywordHost()
  registerKeywordForbiddenCommand(host)

  const session = createSession('1001')
  assert.equal(
    await runCommand(actions, 'forbidden', session, { a: 'spam, ads' }),
    '已经添加了关键词：spam、ads 喵喵喵~',
  )
  assert.deepEqual(groupConfigs.get('1001')?.keywords, ['spam', 'ads'])

  assert.equal(await runCommand(actions, 'forbidden', session, { b: 'on' }), '自动禁言状态更新为true')
  assert.equal(await runCommand(actions, 'forbidden', session, { t: '30m' }), '禁言时间已更新为：30m 喵喵喵~')
  assert.equal(await runCommand(actions, 'forbidden', session, { echo: 'true' }), '回显状态更新为true')

  const groupForbidden = groupConfigs.get('1001')?.forbidden
  assert.equal(groupForbidden?.autoBan, true)
  assert.equal(groupForbidden?.muteDuration, 30 * 60 * 1000)
  assert.equal(groupForbidden?.echo, true)

  const listMessage = await runCommand(actions, 'forbidden', session, { l: true })
  assert.match(listMessage, /全局禁言关键词：\nglobal/)
  assert.match(listMessage, /当前群禁言关键词：\nspam、ads/)
  assert.match(listMessage, /回显状态：开启/)
  assert.match(listMessage, /自动禁言状态：开启/)
})

test('verify command updates typed admission keyword options', async () => {
  const { actions, groupConfigs, host } = createKeywordHost()
  registerKeywordVerifyCommand(host)

  const session = createSession('2001')
  assert.equal(
    await runCommand(actions, 'verify', session, { a: 'student, freshman' }),
    '已经添加了关键词：student、freshman 喵喵喵~',
  )
  assert.deepEqual(groupConfigs.get('2001')?.approvalKeywords, ['student', 'freshman'])

  assert.equal(await runCommand(actions, 'verify', session, { n: 'yes' }), '自动拒绝状态更新为true')
  assert.equal(await runCommand(actions, 'verify', session, { w: '请填写校园邮箱' }), '拒绝词已更新为：请填写校园邮箱 喵喵喵~')

  const groupConfig = groupConfigs.get('2001')
  assert.equal(groupConfig?.auto, 'true')
  assert.equal(groupConfig?.reject, '请填写校园邮箱')

  const listMessage = await runCommand(actions, 'verify', session, { l: true })
  assert.match(listMessage, /当前群入群审核关键词：\nstudent、freshman/)
  assert.match(listMessage, /自动拒绝状态：true/)
  assert.match(listMessage, /拒绝词：请填写校园邮箱/)
})

function readModuleFile(relativePath: string): string {
  return readFileSync(resolve(modulesDir, relativePath), 'utf8')
}

function createKeywordHost(): {
  actions: Map<string, CommandAction>
  groupConfigs: Map<string, GroupConfig>
  host: KeywordModule
} {
  const actions = new Map<string, CommandAction>()
  const groupConfigs = new Map<string, GroupConfig>()

  const host = {
    config: {
      forbidden: {
        autoDelete: false,
        autoBan: false,
        autoKick: false,
        muteDuration: 60_000,
        keywords: ['global'],
      },
    } as Config,
    data: {
      groupConfig: {
        get(guildId: string) {
          return groupConfigs.get(guildId)
        },
        set(guildId: string, config: GroupConfig) {
          groupConfigs.set(guildId, config)
        },
        flush() {
          return undefined
        },
      },
    },
    log: async () => undefined,
    registerCommand(def: { readonly name: string }) {
      const chain: CapturedCommandChain = {
        option: () => chain,
        action(fn: CommandAction) {
          actions.set(def.name, fn)
          return chain
        },
      }
      return chain as unknown as ReturnType<KeywordModule['registerCommand']>
    },
  }

  return {
    actions,
    groupConfigs,
    host: host as unknown as KeywordModule,
  }
}

function createSession(guildId: string): Session {
  return {
    guildId,
    userId: 'operator',
    username: 'Operator',
  } as Session
}

async function runCommand(
  actions: Map<string, CommandAction>,
  name: string,
  session: Session,
  options: Record<string, unknown>,
): Promise<string> {
  const action = actions.get(name)
  if (!action) throw new Error(`Command action not registered: ${name}`)
  return action({ session, options })
}
