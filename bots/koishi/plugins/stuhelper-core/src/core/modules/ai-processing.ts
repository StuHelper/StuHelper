import type { ChatMessage } from '../../types'
import { getUserContext } from './ai-context'
import type { AIModule } from './ai.module'

const DEFAULT_MODEL = 'gpt-3.5-turbo'
const DEFAULT_API_URL = 'https://api.openai.com/v1'
const DEFAULT_SYSTEM_PROMPT = '你是一个有帮助的AI助手。'
const DEFAULT_TRANSLATE_PROMPT =
  '你是一个翻译助手。请将用户的文本准确翻译，保持原文的风格和语气。如果是中文则翻译为英文，如果是其他语言则翻译为中文。不要添加任何解释或额外内容。'
const DEFAULT_TEMPERATURE = 0.7
const TRANSLATE_TEMPERATURE = 0.3
const DEFAULT_MAX_TOKENS = 2048
const MODERATION_MAX_TOKENS = 4096
const DEFAULT_CONTEXT_LIMIT = 10

interface ProcessMessageInput {
  readonly userId: string
  readonly content: string
  readonly guildId?: string
}

export interface TranslateTextInput {
  readonly userId: string
  readonly text: string
  readonly guildId?: string
  readonly customPrompt?: string
}

export async function processAiMessage(host: AIModule, input: ProcessMessageInput): Promise<string> {
  const config = host.config.openai
  if (!config?.enabled) return '抱歉，AI功能当前已禁用。'

  try {
    const systemPrompt = resolveChatPrompt(host, input.guildId)
    if (typeof systemPrompt !== 'string') return systemPrompt.message

    host.addMessageToContext({
      userId: input.userId,
      message: { role: 'user', content: input.content },
      systemPrompt,
      contextLimit: config.contextLimit || DEFAULT_CONTEXT_LIMIT,
    })

    const context = getUserContext(host, input.userId, systemPrompt)
    const response = await host.callOpenAI({
      messages: context.messages,
      model: config.model || DEFAULT_MODEL,
      temperature: config.temperature || DEFAULT_TEMPERATURE,
      maxTokens: config.maxTokens || DEFAULT_MAX_TOKENS,
      apiKey: config.apiKey,
      apiUrl: config.apiUrl || DEFAULT_API_URL,
    })

    const assistantMessage = response.choices[0].message
    host.addMessageToContext({
      userId: input.userId,
      message: assistantMessage,
      systemPrompt,
      contextLimit: config.contextLimit || DEFAULT_CONTEXT_LIMIT,
    })
    return assistantMessage.content
  } catch (error: any) {
    host.data.writeLog(`[ai] AI处理消息失败: ${error}`)
    return `处理消息时出错: ${error.message}`
  }
}

export async function translateAiText(host: AIModule, input: TranslateTextInput): Promise<string> {
  const config = host.config.openai
  if (!config?.enabled) return '抱歉，AI翻译功能当前已禁用。'

  try {
    const translatePrompt = resolveTranslatePrompt(host, input)
    if (typeof translatePrompt !== 'string') return translatePrompt.message

    const messages: ChatMessage[] = [
      { role: 'system', content: translatePrompt },
      { role: 'user', content: input.text },
    ]
    const response = await host.callOpenAI({
      messages,
      model: config.model || DEFAULT_MODEL,
      temperature: TRANSLATE_TEMPERATURE,
      maxTokens: config.maxTokens || DEFAULT_MAX_TOKENS,
      apiKey: config.apiKey,
      apiUrl: config.apiUrl || DEFAULT_API_URL,
    })
    return response.choices[0].message.content
  } catch (error: any) {
    host.data.writeLog(`[ai] AI翻译失败: ${error}`)
    return `翻译出错: ${error.message}`
  }
}

export async function callAiModeration(host: AIModule, prompt: string): Promise<string> {
  const config = host.config.openai
  if (!config?.enabled) throw new Error('AI功能当前已禁用')
  if (!config.apiKey) throw new Error('未配置 API Key')

  try {
    const response = await host.callOpenAI({
      messages: [{ role: 'user', content: prompt }],
      model: config.model || DEFAULT_MODEL,
      temperature: TRANSLATE_TEMPERATURE,
      maxTokens: config.maxTokens || MODERATION_MAX_TOKENS,
      apiKey: config.apiKey,
      apiUrl: config.apiUrl || DEFAULT_API_URL,
    })
    return extractModerationContent(host, response)
  } catch (error: any) {
    host.data.writeLog(`[ai] AI内容审核失败: ${error}`)
    throw new Error(`内容审核失败: ${error.message}`)
  }
}

function resolveChatPrompt(host: AIModule, guildId?: string): string | { message: string } {
  let systemPrompt = host.config.openai.systemPrompt || DEFAULT_SYSTEM_PROMPT
  if (!guildId) return systemPrompt

  const groupConfig = host.getGroupConfig(guildId)
  if (groupConfig?.openai?.enabled === false) return { message: '抱歉，当前群聊已禁用AI功能。' }
  if (groupConfig?.openai?.chatEnabled === false) return { message: '抱歉，当前群聊已禁用AI对话功能。' }
  if (groupConfig?.openai?.systemPrompt) {
    systemPrompt = groupConfig.openai.systemPrompt
  }
  return systemPrompt
}

function resolveTranslatePrompt(host: AIModule, input: TranslateTextInput): string | { message: string } {
  if (input.customPrompt) return input.customPrompt

  let translatePrompt = host.config.openai.translatePrompt || DEFAULT_TRANSLATE_PROMPT
  if (!input.guildId) return translatePrompt

  const groupConfig = host.getGroupConfig(input.guildId)
  if (groupConfig?.openai?.enabled === false) return { message: '抱歉，当前群聊已禁用AI功能。' }
  if (groupConfig?.openai?.translateEnabled === false) return { message: '抱歉，当前群聊已禁用AI翻译功能。' }
  if (groupConfig?.openai?.translatePrompt) {
    translatePrompt = groupConfig.openai.translatePrompt
  }
  return translatePrompt
}

function extractModerationContent(host: AIModule, response: any): string {
  if (!response) throw new Error('API 返回空响应')
  if (!response.choices || !Array.isArray(response.choices) || response.choices.length === 0) {
    host.data.writeLog(`[ai] API 响应格式异常: ${JSON.stringify(response)}`)
    throw new Error('API 响应格式异常，缺少 choices 字段')
  }

  const choice = response.choices[0]
  if (!choice.message || !choice.message.content) {
    host.data.writeLog(`[ai] API 响应缺少内容: ${JSON.stringify(choice)}`)
    throw new Error('API 响应缺少 message.content')
  }
  return choice.message.content
}
