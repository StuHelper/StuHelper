import type { Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { DiceConfig, GroupConfig } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'

const DEFAULT_DICE_LENGTH_LIMIT = 1000
const MIN_DICE_SIDES = 2
const MIN_DICE_COUNT = 1
const DICE_EXPRESSION_PATTERN = /^(\d*)d(\d+)$/i
const ENABLED_VALUES = new Set(['true', '1', 'yes', 'y', 'on'])
const DISABLED_VALUES = new Set(['false', '0', 'no', 'n', 'off'])

type DiceConfigOptions = {
  readonly e?: unknown
  readonly l?: unknown
}

type CommandLogRecord = {
  readonly session: Session
  readonly command: string
  readonly target: string
  readonly result: string
}

type DiceConfigUpdate = {
  readonly session: Session
  readonly configs: Record<string, GroupConfig>
  readonly diceConfig: DiceConfig
  readonly rawValue: unknown
}

/**
 * 骰子游戏模块
 * 支持掷骰子功能，支持 XdY 语法
 */
export class DiceModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'dice',
    description: '骰子游戏模块',
    version: '1.0.0',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    private readonly ctx: Context,
    private readonly data: DataManager,
  ) {}

  get state(): RuntimeModuleState {
    return this._state
  }

  get error(): Error | null {
    return this._error
  }

  async init(): Promise<void> {
    this._state = 'loading'
    try {
      this.registerCommands()
      this.registerMiddleware()
      this.ctx.logger.info('[DiceModule] initialized')
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    this._state = 'unloaded'
  }

  private registerCommands(): void {
    this.registerConfigCommand()
    this.registerRollCommand()
  }

  private registerConfigCommand(): void {
    registerRuntimeCommand(this.ctx, this.meta, {
      name: 'dice-config',
      desc: '掷骰子功能开关',
      permNode: 'dice-config',
      permDesc: '配置掷骰子功能',
      usage: '-e true/false 启用禁用，-l 数字 设置结果长度限制',
      examples: ['dice-config -e true', 'dice-config -l 500'],
    })
      .option('e', '-e <enabled:string> 启用或禁用掷骰子功能')
      .option('l', '-l <length:number> 设置掷骰子结果长度限制')
      .action(async ({ session, options }) => {
        if (!session.guildId) return '此命令只能在群聊中使用。'
        return this.handleConfigCommand(session, options)
      })
  }

  private handleConfigCommand(session: Session, options: DiceConfigOptions): string {
    const { configs, diceConfig } = this.ensureGuildDiceConfig(session.guildId!)

    if (options.e !== undefined) {
      return this.updateDiceEnabled({ session, configs, diceConfig, rawValue: options.e })
    }
    if (options.l !== undefined) {
      return this.updateDiceLength({ session, configs, diceConfig, rawValue: options.l })
    }

    return '请输入要配置的选项，如 -e true 或 -l 1000。'
  }

  private updateDiceEnabled(update: DiceConfigUpdate): string {
    const enabled = parseEnabledOption(update.rawValue)
    if (enabled === undefined) {
      this.recordCommandLog({
        session: update.session,
        command: 'dice-enabled',
        target: update.session.guildId!,
        result: '失败：设置无效',
      })
      return '掷骰子选项无效，请输入 true/false'
    }

    update.diceConfig.enabled = enabled
    this.data.groupConfig.setAll(update.configs)
    const result = enabled ? '成功：已启用掷骰子功能' : '成功：已禁用掷骰子功能'
    this.recordCommandLog({
      session: update.session,
      command: 'dice-enabled',
      target: update.session.guildId!,
      result,
    })
    return enabled ? '掷骰子功能已启用喵~' : '掷骰子功能已禁用喵~'
  }

  private updateDiceLength(update: DiceConfigUpdate): string {
    const length = Number(update.rawValue)
    if (Number.isNaN(length) || length < MIN_DICE_COUNT) {
      return '长度限制必须是大于0的数字。'
    }

    update.diceConfig.lengthLimit = length
    this.data.groupConfig.setAll(update.configs)
    const result = `成功：已设置掷骰子结果长度限制为 ${length}`
    this.recordCommandLog({
      session: update.session,
      command: 'dice-length',
      target: update.session.guildId!,
      result,
    })
    return `已设置掷骰子结果长度限制为 ${length} 喵~`
  }

  private registerRollCommand(): void {
    registerRuntimeCommand(this.ctx, this.meta, {
      name: 'dice',
      desc: '掷骰子',
      args: '<sides:string> [count:string]',
      permNode: 'dice',
      permDesc: '使用掷骰子功能',
      skipAuth: true,
      usage: '掷指定面数的骰子，支持 XdY 语法',
      examples: ['dice 6', 'dice 20 3', '2d6'],
    })
      .example('dice 6')
      .example('dice 20 3')
      .action(async ({ session }, sidesInput, countInput = 1) => {
        if (!session.guildId) return
        return this.handleRollCommand(session.guildId, sidesInput, countInput)
      })
  }

  private handleRollCommand(guildId: string, sidesInput: unknown, countInput: unknown): string {
    const diceConfig = this.getDiceConfig(guildId)
    if (!diceConfig.enabled) return ''

    const sides = Number.parseInt(String(sidesInput))
    const count = Number.parseInt(String(countInput))
    const validationError = validateDiceInput(sides, count)
    if (validationError) return validationError

    if (isDiceResultTooLong(sides, count, getDiceLengthLimit(diceConfig))) {
      return '喵呜...掷骰子结果过长，请选择较少的面数或个数喵~'
    }

    return formatDiceResult('掷骰子结果', this.rollDice(sides, count), '：')
  }

  private registerMiddleware(): void {
    this.ctx.middleware(async (session, next) => {
      if (!session.guildId || !session.content) return next()

      const diceConfig = this.getDiceConfig(session.guildId)
      if (!diceConfig.enabled) return next()

      const match = DICE_EXPRESSION_PATTERN.exec(session.content.trim())
      if (!match) return next()

      const count = Number.parseInt(match[1]) || MIN_DICE_COUNT
      const sides = Number.parseInt(match[2])
      if (sides < MIN_DICE_SIDES || count < MIN_DICE_COUNT) return next()

      if (isDiceResultTooLong(sides, count, getDiceLengthLimit(diceConfig))) {
        await session.send('喵呜...掷骰子结果过长，请选择较少的面数或个数喵~')
        return
      }

      await session.send(formatDiceResult(`掷骰子结果 (${match[0]})`, this.rollDice(sides, count), ': '))
    })
  }

  private ensureGuildDiceConfig(guildId: string) {
    const configs = this.data.groupConfig.getAll() as Record<string, GroupConfig>
    const groupConfig = configs[guildId] ?? {}
    const diceConfig = groupConfig.dice ?? createDefaultDiceConfig()
    configs[guildId] = { ...groupConfig, dice: diceConfig }
    return { configs, diceConfig }
  }

  private getDiceConfig(guildId: string): DiceConfig {
    return this.data.groupConfig.get(guildId)?.dice || createDefaultDiceConfig()
  }

  private recordCommandLog(record: CommandLogRecord): void {
    void this.ctx.stuhelperGroupCenter.logCommand(
      record.session,
      record.command,
      record.target,
      record.result,
    )
  }

  private rollDice(sides: number, count: number): number[] {
    const results: number[] = []
    for (let i = 0; i < count; i++) {
      results.push(Math.floor(Math.random() * sides) + 1)
    }
    return results
  }
}

export const diceRuntimeModule: RuntimeModule<DiceModule> = {
  id: 'dice',
  create(ctx, deps) {
    return new DiceModule(ctx, deps.data)
  },
}

function createDefaultDiceConfig(): Required<DiceConfig> {
  return { enabled: true, lengthLimit: DEFAULT_DICE_LENGTH_LIMIT }
}

function parseEnabledOption(value: unknown): boolean | undefined {
  const normalized = String(value).toLowerCase()
  if (ENABLED_VALUES.has(normalized)) return true
  if (DISABLED_VALUES.has(normalized)) return false
  return undefined
}

function validateDiceInput(sides: number, count: number): string | undefined {
  if (!sides) return '喵呜...请指定骰子面数喵~'
  if (sides < MIN_DICE_SIDES || count < MIN_DICE_COUNT) {
    return '喵呜...骰子面数至少为2，个数至少为1喵~'
  }
  return undefined
}

function isDiceResultTooLong(sides: number, count: number, lengthLimit: number): boolean {
  return (String(sides).length + 2) * count > lengthLimit
}

function getDiceLengthLimit(diceConfig: DiceConfig): number {
  return diceConfig.lengthLimit || DEFAULT_DICE_LENGTH_LIMIT
}

function formatDiceResult(prefix: string, results: number[], separator: string): string {
  if (results.length === 1) return `${prefix}${separator}${results[0]}`

  const sum = results.reduce((a, b) => a + b, 0)
  return `${prefix}${separator}${results.join(', ')}\n总和：${sum}`
}
