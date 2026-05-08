/**
 * HelpModule - 帮助模块
 * 提供帮助信息显示和时间解析测试功能
 */

import { Context } from 'koishi'
import { parseTimeString, formatDuration } from '../../utils'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
const pkg = require('../../../package.json')

export class HelpModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'help',
    description: '帮助模块 - 提供帮助信息和工具命令'
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(private readonly ctx: Context) {}

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
    // 主帮助命令
    registerRuntimeCommand(this.ctx, this.meta, {
      name: 'stuhelperGroupCenter',
      desc: '群管理帮助',
      permNode: 'stuhelperGroupCenter',
      permDesc: '查看帮助信息',
      skipAuth: true
    })
      .option('a', '-a 显示所有可用命令')
      .option('v', '-v 显示当前版本信息')
      .action(async ({ session, options }) => {
        if (options.a) {
          return this.getFullHelpText(session)
        }
        if (options.v) {
          return `当前版本：${pkg.version}`
        }
        return '强大的群管理插件，提供一系列实用的群管理功能\n使用参数 -a 查看所有可用命令'
      })

    // 时间解析测试命令
    registerRuntimeCommand(this.ctx, this.meta, {
      name: 'parse-time',
      desc: '测试时间解析',
      args: '<expression:text>',
      permNode: 'parse-time',
      permDesc: '测试时间解析',
      skipAuth: true
    })
      .example('parse-time 10m')
      .example('parse-time 1h30m')
      .example('parse-time 2days')
      .example('parse-time (1+2)^2hours')
      .action(async ({ session }, expression) => {
        if (!expression) {
          return '请提供时间表达式进行测试'
        }

        try {
          const milliseconds = parseTimeString(expression)
          return `表达式 "${expression}" 解析结果：${formatDuration(milliseconds)} (${milliseconds}毫秒)`
        } catch (e) {
          return `解析错误：${e.message}`
        }
      })
  }

  /**
   * 转义 <> 字符，防止 Koishi 解析为标签
   */
  private escapeAngleBrackets(text: string): string {
    if (!text) return text
    return text.replace(/</g, '&lt;').replace(/>/g, '&gt;')
  }

  private getFullHelpText(session: any): string {
    const commandsByModule = this.ctx.stuhelperGroupCenter.auth.getCommandsByModule()
    const lines: string[] = []

    for (const [moduleDesc, commands] of commandsByModule) {
      const accessibleCommands = commands.filter(cmd => {
        if (!cmd.permId) return true
        return this.ctx.stuhelperGroupCenter.auth.check(session, cmd.permId)
      })

      if (accessibleCommands.length === 0) continue
      lines.push(...this.formatCommandModuleHelp(moduleDesc, accessibleCommands))
    }

    lines.push(this.getTimeExpressionHelp())
    return lines.join('\n')
  }

  private formatCommandModuleHelp(moduleDesc: string, commands: any[]): string[] {
    const lines = [`=== ${moduleDesc} ===`]
    for (const cmd of commands) {
      lines.push(this.formatCommandHelpLine(cmd))
      if (cmd.usage) {
        cmd.usage.split('\n').forEach((line: string) => lines.push(`  ${this.escapeAngleBrackets(line)}`))
      }
      cmd.examples?.forEach((example: string) => lines.push(`  示例：${this.escapeAngleBrackets(example)}`))
    }
    lines.push('')
    return lines
  }

  private formatCommandHelpLine(cmd: any): string {
    const args = cmd.args ? ` ${this.escapeAngleBrackets(cmd.args)}` : ''
    return `${cmd.name}${args}  ${cmd.desc}`
  }

  private getTimeExpressionHelp(): string {
    return `=== 时间表达式 ===
支持以下格式：
· 基本单位：s（秒）、min（分钟）、h（小时）
· 基本格式：数字+单位，如：1h、10min、30s
· 支持的运算：
  · +, -, *, /, ^, sqrt(), 1e2
· 表达式示例：
  · (sqrt(100)+1e1)^2s = 400秒 = 6分40秒
· 时间范围：1秒 ~ 29天23小时59分59秒`
  }
}

export const helpRuntimeModule: RuntimeModule<HelpModule> = {
  id: 'help',
  create(ctx) {
    return new HelpModule(ctx)
  },
}
