import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Session } from 'koishi'

import type { Config, GroupConfig } from '../../types'
import type { AIModule } from './ai.module'
import { registerAiCommands } from './ai-commands'

type CommandAction = (
  argv: {
    readonly session?: Session
    readonly options: Record<string, unknown>
  },
  ...args: string[]
) => string | Promise<string> | undefined

interface CapturedCommandChain {
  alias(...args: unknown[]): CapturedCommandChain
  option(...args: unknown[]): CapturedCommandChain
  example(...args: unknown[]): CapturedCommandChain
  action(fn: CommandAction): CapturedCommandChain
}

interface AiHostOverrides {
  readonly processMessage?: AIModule['processMessage']
}

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('ai command sources use explicit option types and unknown error boundaries', () => {
  const commandsSource = readModuleFile('./ai-commands.ts')
  const middlewareSource = readModuleFile('./ai-middleware.ts')

  assert.doesNotMatch(commandsSource, /options: any/)
  assert.doesNotMatch(commandsSource, /catch \(error: any\)/)
  assert.doesNotMatch(commandsSource, /groupConfigs: any/)
  assert.doesNotMatch(commandsSource, /openaiConfig: any/)
  assert.doesNotMatch(middlewareSource, /session: any/)
  assert.doesNotMatch(middlewareSource, /el: any/)
  assert.match(commandsSource, /interface AiConfigCommandOptions/)
  assert.match(middlewareSource, /function isMentioningBot\(session: Session\)/)
})

test('ai-config command updates and resets typed group OpenAI config', async () => {
  const { actions, groupConfigs, host } = createAiHost()
  registerAiCommands(host)

  const session = createSession('guild-1')
  assert.equal(
    await runCommand(actions, 'ai-config', session, {
      enabled: false,
      prompt: 'group system prompt',
      tprompt: 'group translate prompt',
    }),
    '群AI配置已更新',
  )
  assert.deepEqual(groupConfigs['guild-1'].openai, {
    enabled: false,
    systemPrompt: 'group system prompt',
    translatePrompt: 'group translate prompt',
  })

  const configMessage = await runCommand(actions, 'ai-config', session, {})
  assert.match(configMessage, /AI总开关: 禁用/)
  assert.match(configMessage, /系统提示词: group system prompt/)
  assert.match(configMessage, /翻译提示词: group translate prompt/)

  assert.equal(
    await runCommand(actions, 'ai-config', session, { reset: true }),
    '已重置为全局AI配置',
  )
  assert.equal(groupConfigs['guild-1'].openai, undefined)
})

test('ai command reports non-Error processing failures without losing the cause', async () => {
  const { actions, host } = createAiHost({
    processMessage: async () => Promise.reject('adapter unavailable'),
  })
  registerAiCommands(host)

  assert.equal(
    await runCommand(actions, 'ai', createSession('guild-1'), {}, 'hello'),
    '处理请求时出错: adapter unavailable',
  )
})

function readModuleFile(relativePath: string): string {
  return readFileSync(resolve(modulesDir, relativePath), 'utf8')
}

function createAiHost(overrides: AiHostOverrides = {}): {
  readonly actions: Map<string, CommandAction>
  readonly groupConfigs: Record<string, GroupConfig>
  readonly host: AIModule
} {
  const actions = new Map<string, CommandAction>()
  const groupConfigs: Record<string, GroupConfig> = {}

  const host = {
    config: {
      openai: {
        enabled: true,
      },
    } as Config,
    data: {
      groupConfig: {
        getAll() {
          return groupConfigs
        },
        setAll(data: Record<string, GroupConfig>) {
          if (data === groupConfigs) return
          Object.keys(groupConfigs).forEach((guildId) => {
            delete groupConfigs[guildId]
          })
          Object.assign(groupConfigs, data)
        },
      },
      writeLog: () => undefined,
    },
    resetUserContext: () => false,
    processMessage: overrides.processMessage ?? (async () => 'AI response'),
    translateText: async () => 'translated',
    log: async () => undefined,
    registerCommand(def: { readonly name: string }) {
      const chain: CapturedCommandChain = {
        alias: () => chain,
        option: () => chain,
        example: () => chain,
        action(fn: CommandAction) {
          actions.set(def.name, fn)
          return chain
        },
      }
      return chain as unknown as ReturnType<AIModule['registerCommand']>
    },
  }

  return {
    actions,
    groupConfigs,
    host: host as unknown as AIModule,
  }
}

function createSession(guildId: string): Session {
  return {
    guildId,
    userId: 'operator',
    messageId: 'message-1',
  } as Session
}

async function runCommand(
  actions: Map<string, CommandAction>,
  name: string,
  session: Session,
  options: Record<string, unknown>,
  ...args: string[]
): Promise<string> {
  const action = actions.get(name)
  if (!action) throw new Error(`Command action not registered: ${name}`)
  const result = await action({ session, options }, ...args)
  if (typeof result !== 'string') throw new Error(`Command action did not return a string: ${name}`)
  return result
}
