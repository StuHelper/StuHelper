import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Session } from 'koishi'

import type { Config, WarnRecord } from '../../types'
import type { WarnModule } from './warn.module'
import { registerWarnCommands } from './warn-commands.ts'

type CommandAction = (
  argv: { readonly session?: Session },
  user?: unknown,
  count?: unknown,
) => string | Promise<string>

interface CapturedCommandChain {
  action(fn: CommandAction): CapturedCommandChain
}

interface WarnCommandHostOverrides {
  readonly warnLimit?: number
}

interface WarnSessionOverrides {
  readonly muteGuildMember?: Session['bot']['muteGuildMember']
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('warn command source uses explicit command boundary types', () => {
  const source = readFileSync(resolve(modulesDir, './warn-commands.ts'), 'utf8')

  assert.doesNotMatch(source, /catch \(error: any\)/)
  assert.doesNotMatch(source, /error: any/)
  assert.match(source, /function normalizeWarnCount/)
  assert.match(source, /function errorMessage/)
})

test('warn commands report a missing session instead of dereferencing it', async () => {
  const { actions } = createWarnCommandHost()
  const result = await runWarnCommand(actions, 'warn', undefined, 'target-user', 1)

  assert.equal(result, '无法读取当前会话')
})

test('warn command rejects non-positive and fractional counts before mutating state', async () => {
  const { actions, host, logs, messages } = createWarnCommandHost()
  const session = createWarnSession()

  assert.equal(await runWarnCommand(actions, 'warn', session, 'target-user', -1), '警告次数必须是正整数喵！')
  assert.equal(await runWarnCommand(actions, 'warn', session, 'target-user', 1.5), '警告次数必须是正整数喵！')
  assert.equal(host.data.warns.get('guild-1'), undefined)
  assert.deepEqual(logs, [])
  assert.deepEqual(messages, [])
})

test('warn command handles non-Error auto-ban failures without undefined output', async () => {
  const { actions, host, logs, messages } = createWarnCommandHost({
    warnLimit: 1,
  })
  const session = createWarnSession({
    muteGuildMember: async () => {
      throw 'mute offline'
    },
  })

  const result = await runWarnCommand(actions, 'warn', session, 'target-user', 1)

  assert.equal(host.data.warns.get('guild-1')?.['target-user']?.count, 1)
  assert.equal(result, '警告已记录，但自动禁言失败：mute offline')
  assert.match(messages.at(-1) ?? '', /自动禁言失败：mute offline/)
  assert.equal(logs.at(-1)?.command, 'warn')
  assert.match(logs.at(-1)?.result ?? '', /但自动禁言失败/)
})

function createWarnCommandHost(overrides: WarnCommandHostOverrides = {}): {
  readonly actions: Map<string, CommandAction>
  readonly host: WarnModule
  readonly logs: Array<{ command: string; target: string; result: string }>
  readonly messages: string[]
} {
  const actions = new Map<string, CommandAction>()
  const logs: Array<{ command: string; target: string; result: string }> = []
  const messages: string[] = []
  const warns = createMapStore<WarnRecord>()
  const mutes = createMapStore<Record<string, { startTime: number; duration: number; remainingTime?: number }>>()

  const host = {
    config: {
      warnLimit: overrides.warnLimit ?? 3,
      banTimes: {
        expression: '10m',
      },
    } as Config,
    ctx: {
      stuhelperGroupCenter: {
        pushMessage: async (_bot: Session['bot'], message: string) => {
          messages.push(message)
        },
      },
    },
    data: {
      warns,
      mutes,
    },
    registerCommand(def: { readonly name: string }) {
      const chain: CapturedCommandChain = {
        action(fn: CommandAction) {
          actions.set(def.name, fn)
          return chain
        },
      }
      return chain as unknown as ReturnType<WarnModule['registerCommand']>
    },
    log: async (entry: { readonly command: string; readonly target: string; readonly result: string }) => {
      logs.push(entry)
    },
    getGroupConfig: () => undefined,
    addWarn(guildId: string, userId: string, count: number) {
      const guildWarns = warns.get(guildId) ?? {}
      const record = guildWarns[userId] ?? { count: 0, timestamp: Date.now() }
      record.count += count
      record.timestamp = Date.now()
      guildWarns[userId] = record
      warns.set(guildId, guildWarns)
      warns.flush()
      return record.count
    },
    recordMute(guildId: string, userId: string, duration: number) {
      const guildMutes = mutes.get(guildId) ?? {}
      guildMutes[userId] = { startTime: Date.now(), duration, remainingTime: duration }
      mutes.set(guildId, guildMutes)
      mutes.flush()
    },
  }

  registerWarnCommands(host as unknown as WarnModule)
  return {
    actions,
    host: host as unknown as WarnModule,
    logs,
    messages,
  }
}

function createMapStore<T>(): {
  readonly values: Map<string, T>
  get(key: string): T | undefined
  set(key: string, value: T): void
  delete(key: string): void
  flush(): void
} {
  const values = new Map<string, T>()
  return {
    values,
    get: (key) => values.get(key),
    set: (key, value) => {
      values.set(key, value)
    },
    delete: (key) => {
      values.delete(key)
    },
    flush: () => {},
  }
}

function createWarnSession(overrides: WarnSessionOverrides = {}): Session {
  return {
    guildId: 'guild-1',
    userId: 'operator',
    bot: {
      muteGuildMember: overrides.muteGuildMember ?? (async () => {}),
    },
  } as unknown as Session
}

async function runWarnCommand(
  actions: Map<string, CommandAction>,
  command: string,
  session: Session | undefined,
  user?: unknown,
  count?: unknown,
): Promise<string> {
  const action = actions.get(command)
  if (!action) throw new Error(`${command} action was not registered`)
  return action({ session }, user, count)
}
