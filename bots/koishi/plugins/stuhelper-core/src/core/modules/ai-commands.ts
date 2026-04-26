import type { Session } from 'koishi'
import { h } from 'koishi'

import type { AIModule } from './ai.module'

interface CommandInput {
  host: AIModule
  session: Session
  options: any
}

export function registerAiCommands(host: AIModule): void {
  registerAiChatCommand(host)
  registerTranslateCommand(host)
  registerAiConfigCommand(host)
}

function registerAiChatCommand(host: AIModule): void {
  host.registerCommand({
    name: 'ai',
    desc: '与AI进行对话',
    args: '[content:text]',
    permNode: 'ai',
    permDesc: '使用AI对话功能',
    skipAuth: true,
    usage: '-r 重置对话上下文',
    examples: ['ai 今天天气怎么样', 'ai -r'],
  })
    .option('reset', '-r 重置对话上下文')
    .action(async ({ session, options }, content) => {
      if (!session) return
      return handleAiChatCommand({ host, session, options }, content)
    })
}

async function handleAiChatCommand(input: CommandInput, content?: string): Promise<string> {
  const { host, session, options } = input
  if (!host.config.openai?.enabled) return 'AI功能已被全局禁用'

  if (options?.reset) {
    const reset = host.resetUserContext(session.userId)
    void host.log({ session, command: 'ai', target: 'reset', result: reset ? '成功' : '无上下文' })
    return reset ? '对话上下文已重置' : '没有找到对话上下文'
  }
  if (!content) return '请输入您要问AI的内容，例如：ai 今天天气怎么样？'

  try {
    void host.log({ session, command: 'ai', target: 'chat', result: content.substring(0, 50) })
    return await host.processMessage(session.userId, content, session.guildId)
  } catch (error: any) {
    return `处理请求时出错: ${error.message}`
  }
}

function registerTranslateCommand(host: AIModule): void {
  host.registerCommand({
    name: 'translate',
    desc: '使用AI翻译文本',
    args: '<text:text>',
    permNode: 'translate',
    permDesc: '使用AI翻译功能',
    skipAuth: true,
    usage: '翻译文本，可回复消息翻译，-p 自定义提示词',
    examples: ['tsl Hello World', 'tsl -p 翻译成日语 你好'],
  })
    .alias('tsl')
    .option('prompt', '-p <prompt:text> 自定义翻译提示词')
    .action(async ({ session, options }, text) => {
      if (!session) return
      return handleTranslateCommand({ host, session, options }, text)
    })
}

async function handleTranslateCommand(input: CommandInput, rawText?: string): Promise<string> {
  const { host, session, options } = input
  if (!host.config.openai?.enabled) return 'AI功能已被全局禁用'

  const text = rawText || session.quote?.content
  if (!text) return '请提供要翻译的文本，或回复需要翻译的消息。\n用法：tsl [文本] 或 回复消息后使用 tsl'

  try {
    void host.log({ session, command: 'translate', target: 'request', result: text.substring(0, 50) })
    const translatedText = await host.translateText({
      userId: session.userId,
      text,
      guildId: session.guildId,
      customPrompt: options?.prompt,
    })
    return session.guildId && session.messageId
      ? h.quote(session.messageId) + translatedText
      : translatedText
  } catch (error: any) {
    return `翻译时出错: ${error.message}`
  }
}

function registerAiConfigCommand(host: AIModule): void {
  host.registerCommand({
    name: 'ai-config',
    desc: '配置AI功能',
    permNode: 'ai-config',
    permDesc: '配置AI功能',
    usage: '-e 启用/禁用，-p 系统提示词，-tp 翻译提示词，-r 重置',
  })
    .option('enabled', '-e <enabled:boolean> 是否在本群启用AI功能')
    .option('prompt', '-p <prompt:text> 设置本群特定的系统提示词')
    .option('tprompt', '-tp <prompt:text> 设置本群特定的翻译提示词')
    .option('reset', '-r 重置为全局配置')
    .action(async ({ session, options }) => {
      if (!session?.guildId) return '此命令只能在群聊中使用'
      return handleAiConfigCommand({ host, session, options })
    })
}

async function handleAiConfigCommand(input: CommandInput): Promise<string> {
  const { host, session, options } = input
  const groupConfigs = host.data.groupConfig.getAll()
  groupConfigs[session.guildId] = groupConfigs[session.guildId] || {}

  if (options?.reset) return resetAiConfig(input, groupConfigs)
  if (!groupConfigs[session.guildId].openai) {
    groupConfigs[session.guildId].openai = { enabled: true }
  }

  const hasChanges = applyAiConfigOptions(input, groupConfigs)
  if (!hasChanges) return formatAiConfig(groupConfigs[session.guildId].openai)

  host.data.groupConfig.setAll(groupConfigs)
  void host.log({ session, command: 'ai-config', target: 'update', result: '成功' })
  return '群AI配置已更新'
}

function resetAiConfig(input: CommandInput, groupConfigs: any): string {
  const { host, session } = input
  if (!groupConfigs[session.guildId].openai) return '本群未设置特定AI配置'

  delete groupConfigs[session.guildId].openai
  host.data.groupConfig.setAll(groupConfigs)
  void host.log({ session, command: 'ai-config', target: 'reset', result: '成功' })
  return '已重置为全局AI配置'
}

function applyAiConfigOptions(input: CommandInput, groupConfigs: any): boolean {
  const openai = groupConfigs[input.session.guildId].openai!
  let hasChanges = false
  if (input.options?.enabled !== undefined) {
    openai.enabled = input.options.enabled
    hasChanges = true
  }
  if (input.options?.prompt) {
    openai.systemPrompt = input.options.prompt
    hasChanges = true
  }
  if (input.options?.tprompt) {
    openai.translatePrompt = input.options.tprompt
    hasChanges = true
  }
  return hasChanges
}

function formatAiConfig(openaiConfig: any): string {
  return [
    '当前群AI配置：',
    `AI总开关: ${openaiConfig?.enabled === undefined ? '跟随全局' : (openaiConfig.enabled ? '启用' : '禁用')}`,
    `系统提示词: ${openaiConfig?.systemPrompt || '跟随全局'}`,
    `翻译提示词: ${openaiConfig?.translatePrompt || '跟随全局'}`,
  ].join('\n')
}
