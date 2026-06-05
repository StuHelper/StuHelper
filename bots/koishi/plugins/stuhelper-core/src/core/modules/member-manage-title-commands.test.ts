import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Session } from 'koishi'

import type { Config } from '../../types'
import type { MemberManageModule } from './memberManage.module'
import { registerTitleCommand } from './member-manage-title-commands.ts'

type CommandAction = (argv: {
  readonly session?: Session
  readonly options?: unknown
}) => string | Promise<string>

interface CapturedCommandChain {
  option(...args: unknown[]): CapturedCommandChain
  action(fn: CommandAction): CapturedCommandChain
}

interface TitleSessionOverrides {
  readonly setGroupSpecialTitle?: (guildId: string, userId: string, title: string) => Promise<unknown>
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('title command source keeps explicit unknown boundaries', () => {
  const source = readFileSync(resolve(modulesDir, './member-manage-title-commands.ts'), 'utf8')
  const moduleSource = readFileSync(resolve(modulesDir, './memberManage.module.ts'), 'utf8')

  assert.doesNotMatch(source, /options: any/)
  assert.doesNotMatch(source, /出错啦喵\.\.\.\$\{error\.message\}/)
  assert.doesNotMatch(moduleSource, /readonly session: any/)
  assert.match(source, /interface TitleCommandOptions/)
  assert.match(source, /function normalizeTitleOptions/)
})

test('title command reports a missing session instead of dereferencing it', async () => {
  const { actions } = createTitleCommandHost()

  assert.equal(await runTitleCommand(actions, undefined, { s: '新头衔' }), '无法读取当前会话')
})

test('title command ignores malformed set-title options and avoids writing literal true', async () => {
  const calls: Array<[string, string, string]> = []
  const { actions, logs } = createTitleCommandHost()
  const session = createTitleSession({
    setGroupSpecialTitle: async (...args) => {
      calls.push(args)
    },
  })

  const result = await runTitleCommand(actions, session, { s: true })

  assert.equal(result, '请使用 -s <文本> 设置头衔或 -r 移除头衔\n可选 -u @用户 为指定用户设置')
  assert.deepEqual(calls, [])
  assert.deepEqual(logs, [])
})

test('title command normalizes target options and uses the verified guild id', async () => {
  const calls: Array<[string, string, string]> = []
  const { actions, logs } = createTitleCommandHost()
  const session = createTitleSession({
    setGroupSpecialTitle: async (...args) => {
      calls.push(args)
    },
  })

  const result = await runTitleCommand(actions, session, { s: '  风纪委员  ', u: 'qq:target-user' })

  assert.equal(result, '已经设置好头衔啦喵~')
  assert.deepEqual(calls, [['guild-1', 'target-user', '风纪委员']])
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'title',
    target: 'target-user',
    result: '成功：已设置头衔：风纪委员',
  })
})

test('title command reports non-Error adapter failures without undefined output', async () => {
  const { actions, logs } = createTitleCommandHost()
  const session = createTitleSession({
    setGroupSpecialTitle: async () => {
      throw 'adapter offline'
    },
  })

  const result = await runTitleCommand(actions, session, { s: '风纪委员' })

  assert.equal(result, '出错啦喵...adapter offline')
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'title',
    target: 'operator',
    result: '失败：未知错误',
    success: false,
  })
})

function createTitleCommandHost(): {
  readonly actions: Map<string, CommandAction>
  readonly logs: Array<{ command: string; target: string; result: string; success?: boolean; session: Session }>
} {
  const actions = new Map<string, CommandAction>()
  const logs: Array<{ command: string; target: string; result: string; success?: boolean; session: Session }> = []
  const host = {
    config: {
      setTitle: {
        enabled: true,
        authority: 3,
        maxLength: 18,
      },
    } as Config,
    registerCommand(def: { readonly name: string }) {
      assert.equal(def.name, 'title')
      const chain: CapturedCommandChain = {
        option: () => chain,
        action(fn: CommandAction) {
          actions.set(def.name, fn)
          return chain
        },
      }
      return chain as unknown as ReturnType<MemberManageModule['registerCommand']>
    },
    logCommand(entry: { readonly command: string; readonly target: string; readonly result: string; readonly success?: boolean; readonly session: Session }) {
      logs.push(entry)
    },
  }

  registerTitleCommand(host as unknown as MemberManageModule)
  return { actions, logs }
}

function createTitleSession(overrides: TitleSessionOverrides = {}): Session {
  return {
    platform: 'qq',
    guildId: 'guild-1',
    userId: 'operator',
    bot: {
      platform: 'onebot',
      internal: {
        setGroupSpecialTitle: overrides.setGroupSpecialTitle ?? (async () => {}),
      },
    },
  } as unknown as Session
}

async function runTitleCommand(
  actions: Map<string, CommandAction>,
  session: Session | undefined,
  options: unknown,
): Promise<string> {
  const action = actions.get('title')
  if (!action) throw new Error('title action was not registered')
  return action({ session, options })
}
