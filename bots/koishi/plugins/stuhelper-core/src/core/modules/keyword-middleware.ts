import type { Session } from 'koishi'

import type { Config, GroupConfig } from '../../types'
import { formatDuration } from '../../utils'
import type { KeywordModule } from './keyword.module'

type GroupForbiddenConfig = NonNullable<GroupConfig['forbidden']>
type EffectiveForbiddenConfig = Config['forbidden'] & Pick<GroupForbiddenConfig, 'echo'>

interface KeywordMatchInput {
  host: KeywordModule
  session: Session
  content: string
  keywords: string[]
  forbiddenConfig: EffectiveForbiddenConfig
}

interface KeywordActionInput {
  host: KeywordModule
  session: Session
  forbiddenConfig: EffectiveForbiddenConfig
}

export function registerKeywordMiddleware(host: KeywordModule): void {
  host.ctx.middleware(async (session, next) => {
    if (!session.content || !session.guildId) return next()

    const content = sanitizeMessageContent(session.content)
    if (!content) return next()

    const groupConfig = host.data.groupConfig.get(session.guildId) || {} as GroupConfig
    const forbiddenConfig = getEffectiveForbiddenConfig(host.config, groupConfig)
    const effectiveKeywords = [...host.config.forbidden.keywords, ...(groupConfig.keywords || [])]
    if (effectiveKeywords.length === 0) return next()

    const input = { host, session, content, keywords: effectiveKeywords, forbiddenConfig }
    if (forbiddenConfig.autoDelete) await handleAutoDelete(input)
    if (forbiddenConfig.autoBan && await handleAutoBan(input)) return

    return next()
  })
}

function getEffectiveForbiddenConfig(config: Config, groupConfig: GroupConfig): EffectiveForbiddenConfig {
  return { ...config.forbidden, ...(groupConfig.forbidden || {}) }
}

function sanitizeMessageContent(content: string): string {
  return content
    .replace(/<at id="\d+"\/>/g, '')
    .replace(/<img[^>]+>/g, '')
    .trim()
}

async function handleAutoDelete(input: KeywordMatchInput): Promise<void> {
  const keyword = findMatchedKeyword(input.host, input.content, input.keywords)
  if (!keyword) return

  try {
    await input.session.bot.deleteMessage(input.session.guildId, input.session.messageId)
    void input.host.log({
      session: input.session,
      command: 'keyword-delete',
      target: input.session.userId,
      result: '成功：关键词匹配，消息已撤回',
    })
    if (input.forbiddenConfig.echo) {
      await input.session.send('喵呜！发现了关键词，消息已被撤回...')
    }
  } catch {
    void input.host.log({ session: input.session, command: 'keyword-delete', target: input.session.userId, result: '失败' })
    if (input.forbiddenConfig.echo) {
      await input.session.send('自动撤回失败了...可能是权限不够喵')
    }
  }
}

async function handleAutoBan(input: KeywordMatchInput): Promise<boolean> {
  const keyword = findMatchedKeyword(input.host, input.content, input.keywords)
  if (!keyword) return false

  if (input.forbiddenConfig.autoKick && await handleKeywordKick(input)) {
    return true
  }
  return handleKeywordMute(input)
}

async function handleKeywordKick(input: KeywordMatchInput): Promise<boolean> {
  try {
    await input.session.bot.kickGuildMember(input.session.guildId, input.session.userId)
    void input.host.log({
      session: input.session,
      command: 'keyword-kick',
      target: input.session.userId,
      result: '成功：关键词匹配，已踢出群聊',
    })
    await input.session.send(`喵呜！发现了关键词，${input.session.username} 已被踢出群聊...`)
    return true
  } catch {
    void input.host.log({ session: input.session, command: 'keyword-kick', target: input.session.userId, result: '失败' })
    await input.session.send('自动踢出失败了...可能是权限不够喵')
    return false
  }
}

async function handleKeywordMute(input: KeywordMatchInput): Promise<boolean> {
  const actionInput = {
    host: input.host,
    session: input.session,
    forbiddenConfig: input.forbiddenConfig,
  }
  try {
    const result = getKeywordMuteDuration(actionInput)
    await input.session.bot.muteGuildMember(input.session.guildId, input.session.userId, result.duration)
    input.host.recordMute(input.session.guildId, input.session.userId, result.duration)
    await sendMuteResult(actionInput, result)
    return true
  } catch {
    void input.host.log({ session: input.session, command: 'keyword-ban', target: input.session.userId, result: '失败' })
    if (input.forbiddenConfig.echo) {
      await input.session.send('自动禁言失败了...可能是权限不够喵')
    }
    return false
  }
}

function getKeywordMuteDuration(input: KeywordActionInput) {
  let duration = input.forbiddenConfig.muteDuration
  const guildMutes = input.host.data.mutes.get(input.session.guildId) || {}
  const lastMute = guildMutes[input.session.userId]
  if (lastMute && lastMute.startTime + lastMute.duration > Date.now() + duration) {
    duration = lastMute.startTime + lastMute.duration - Date.now()
    return { duration, covered: true }
  }
  return { duration, covered: false }
}

async function sendMuteResult(input: KeywordActionInput, result: { duration: number; covered: boolean }): Promise<void> {
  const duration = formatDuration(result.duration)
  const logText = result.covered
    ? `成功：关键词匹配，已有更长禁言，禁言时长 ${duration}`
    : `成功：关键词匹配，禁言时长 ${duration}`
  void input.host.log({ session: input.session, command: 'keyword-ban', target: input.session.userId, result: logText })

  if (!input.forbiddenConfig.echo) return
  const message = result.covered
    ? `喵呜！发现了关键词，检测到未完成的禁言，要被禁言 ${duration} 啦...`
    : `喵呜！发现了关键词，要被禁言 ${duration} 啦...`
  await input.session.send(message)
}

function findMatchedKeyword(host: KeywordModule, content: string, keywords: string[]): string | null {
  for (const keyword of keywords) {
    if (host.matchKeyword(content, keyword)) return keyword
  }
  return null
}
