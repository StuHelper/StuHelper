import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Context, Session } from 'koishi'

import type { BanMeConfig, Config, GroupConfig } from '../../types'
import type { DataManager } from '../data'
import type { BanmeCommandHost } from './banme-commands.ts'
import { registerBanmeCommands } from './banme-commands.ts'
import { BanmeModule } from './banme.module'

type CommandAction = (
  argv: { readonly session?: Session; readonly options?: unknown },
  ...args: unknown[]
) => string | Promise<string | null> | null

interface CapturedCommandChain {
  option(...args: unknown[]): CapturedCommandChain
  action(fn: CommandAction): CapturedCommandChain
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('banme command source uses explicit config option boundaries', () => {
  const source = readFileSync(resolve(modulesDir, './banme-commands.ts'), 'utf8')
  const moduleSource = readFileSync(resolve(modulesDir, './banme.module.ts'), 'utf8')

  assert.doesNotMatch(source, /options: any/)
  assert.doesNotMatch(source, /handleConfigCommand\(host: BanmeCommandHost, session: Session, options: any\)/)
  assert.doesNotMatch(source, /applyNumericOptions\(config: BanMeConfig, options: any\)/)
  assert.doesNotMatch(source, /applyDurationOptions\(config: BanMeConfig, options: any\)/)
  assert.doesNotMatch(moduleSource, /\(error as Error\)\.message/)
  assert.match(source, /interface BanmeConfigCommandOptions/)
  assert.match(source, /function normalizeConfigOptions/)
  assert.match(moduleSource, /commandErrorMessage/)
})

test('banme config command reports a missing session instead of dereferencing it', async () => {
  const { actions } = createBanmeCommandHost()

  assert.equal(await runConfigCommand(actions, undefined, { enabled: true }), '无法读取当前会话')
})

test('banme config rejects malformed numeric options without mutating group config', async () => {
  const { actions, groupConfigs, logs } = createBanmeCommandHost()
  const session = createSession()

  const result = await runConfigCommand(actions, session, { baseMin: '10' })

  assert.equal(result, '最小禁言时间必须是正数')
  assert.deepEqual(groupConfigs.getAll(), {})
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'banme.config',
    target: 'operator',
    result: '失败：最小禁言时间必须是正数',
  })
})

test('banme config preserves valid zero probability instead of dropping it as falsy', async () => {
  const { actions, groupConfigs, logs } = createBanmeCommandHost()
  const session = createSession()

  const result = await runConfigCommand(actions, session, { prob: 0, baseMin: 30, baseMax: 1 })

  assert.equal(result, '配置已更新喵~')
  assert.equal(groupConfigs.getAll()['guild-1']?.banme?.jackpot.baseProb, 0)
  assert.equal(groupConfigs.getAll()['guild-1']?.banme?.baseMin, 30)
  assert.equal(groupConfigs.getAll()['guild-1']?.banme?.baseMax, 1)
  assert.equal(logs.at(-1)?.result, '成功：更新banme配置')
})

test('banme config rejects impossible ranges without storing a partial draft', async () => {
  const { actions, groupConfigs, logs } = createBanmeCommandHost()
  const session = createSession()

  const result = await runConfigCommand(actions, session, { baseMin: 120, baseMax: 1 })

  assert.equal(result, '最小禁言时间不能大于最大禁言时间')
  assert.deepEqual(groupConfigs.getAll(), {})
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'banme.config',
    target: 'operator',
    result: '失败：最小禁言时间不能大于最大禁言时间',
  })
})

test('banme config rejects invalid jackpot durations before storing config', async () => {
  const { actions, groupConfigs } = createBanmeCommandHost()
  const session = createSession()

  const result = await runConfigCommand(actions, session, { uptime: 'later' })

  assert.equal(result, 'UP奖励时长格式无效')
  assert.deepEqual(groupConfigs.getAll(), {})
})

test('banme config only treats a real reset flag as reset and trims boolean text', async () => {
  const existing = createDefaultBanmeConfig()
  const { actions, groupConfigs } = createBanmeCommandHost({
    'guild-1': {
      banme: existing,
    },
  })
  const session = createSession()

  const result = await runConfigCommand(actions, session, { reset: 'false', enabled: ' false ' })

  assert.equal(result, '配置已更新喵~')
  assert.equal(groupConfigs.getAll()['guild-1']?.banme?.enabled, false)
})

test('banme command reports non-Error mute failures without undefined output', async () => {
  const { module, logs } = createBanmeModule()
  const session = {
    guildId: 'guild-1',
    userId: 'operator',
    username: 'operator',
    bot: {
      muteGuildMember: async () => {
        throw 'adapter offline'
      },
    },
  } as unknown as Session

  const result = await module.executeBanme(session)

  assert.equal(result, '喵呜...禁言失败了：adapter offline')
  assert.deepEqual(logs.at(-1), {
    session,
    command: 'banme',
    target: 'operator',
    result: '失败：未知错误',
  })
})

function createBanmeModule(): {
  readonly module: BanmeModule
  readonly logs: Array<{ session: Session; command: string; target: string; result: string }>
} {
  const logs: Array<{ session: Session; command: string; target: string; result: string }> = []
  const ctx = {
    logger: {
      info: () => {},
      error: () => {},
    },
    stuhelperGroupCenter: {
      pluginConfig: {
        banme: createDefaultBanmeConfig(),
      },
      logCommand: async (entry: { session: Session; command: string; target: string; result: string }) => {
        logs.push(entry)
      },
    },
    middleware: () => {},
  }
  const data = {
    groupConfig: new MemoryStore<Record<string, GroupConfig>>({}),
    banmeRecords: new MemoryStore({}),
    mutes: new MemoryStore({}),
  } as unknown as DataManager

  return { module: new BanmeModule(ctx as unknown as Context, data), logs }
}

function createBanmeCommandHost(initialGroupConfigs: Record<string, GroupConfig> = {}): {
  readonly actions: Map<string, CommandAction>
  readonly groupConfigs: MemoryStore<Record<string, GroupConfig>>
  readonly logs: Array<{ session: Session; command: string; target: string; result: string }>
} {
  const actions = new Map<string, CommandAction>()
  const logs: Array<{ session: Session; command: string; target: string; result: string }> = []
  const groupConfigs = new MemoryStore<Record<string, GroupConfig>>(initialGroupConfigs)

  const host: BanmeCommandHost = {
    data: {
      groupConfig: groupConfigs,
    } as unknown as DataManager,
    config: {
      banme: createDefaultBanmeConfig(),
    } as Config,
    registerCommand(def) {
      const chain: CapturedCommandChain = {
        option: () => chain,
        action(fn: CommandAction) {
          actions.set(def.name, fn)
          return chain
        },
      }
      return chain as ReturnType<BanmeCommandHost['registerCommand']>
    },
    executeBanme: async () => null,
    normalizeCommand: (command) => command,
    readSimilarChars: () => null,
    saveSimilarChars: () => {},
    setDefaultSimilarChars: () => {},
    log: async (entry) => {
      logs.push(entry)
    },
  }

  registerBanmeCommands(host)
  return { actions, groupConfigs, logs }
}

function createDefaultBanmeConfig(): BanMeConfig {
  return {
    enabled: true,
    baseMin: 10,
    baseMax: 10,
    growthRate: 1,
    autoBan: false,
    jackpot: {
      enabled: true,
      baseProb: 0.006,
      softPity: 74,
      hardPity: 90,
      upDuration: '7d',
      loseDuration: '1d',
    },
  }
}

function createSession(): Session {
  return {
    guildId: 'guild-1',
    userId: 'operator',
  } as unknown as Session
}

async function runConfigCommand(
  actions: Map<string, CommandAction>,
  session: Session | undefined,
  options: unknown,
): Promise<string | null> {
  const action = actions.get('banme.config')
  if (!action) throw new Error('banme.config action was not registered')
  return action({ session, options })
}

class MemoryStore<T extends Record<string, unknown>> {
  constructor(private data: T) {}

  getAll(): T {
    return this.data
  }

  get<K extends keyof T>(key: K): T[K] | undefined {
    return this.data[key]
  }

  setAll(data: T): void {
    this.data = data
  }
}
