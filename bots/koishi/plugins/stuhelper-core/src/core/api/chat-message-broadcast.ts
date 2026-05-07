import { h, type Context } from 'koishi'

import type { StuhelperGroupCenterService } from '../services/stuhelper-group-center.service'
import type { ChatImageAccessRegistry } from './chat-image-fetch'
import { prependQuoteElement, serializeChatElements } from './chat-element-serializer'
import { deliverChatMessageToClients } from './chat-delivery'
import { toApiErrorMessage } from './api-response'

interface BroadcastOptions {
  readonly ctx: Context
  readonly service: StuhelperGroupCenterService
  readonly chatImageAccess: ChatImageAccessRegistry
}

const QQ_PLATFORMS = ['onebot', 'red', 'qq']

export function registerChatMessageBroadcast(options: BroadcastOptions) {
  const botLoginInfoCache = new Map<string, { userId: string; nickname: string }>()
  const broadcast = (session: any, isSelf = false) => broadcastMessage({
    ...options,
    botLoginInfoCache,
    session,
    isSelf,
  })
  options.ctx.on('message', (session) => broadcast(session))
  options.ctx.logger('stuhelperGroupCenter').info('Chat message listener registered')
  options.ctx.on('send', (session) => broadcast(session, true))
}

async function broadcastMessage(input: BroadcastInput) {
  const { ctx, session, isSelf } = input
  ctx.logger('stuhelperGroupCenter').debug('broadcastMessage called:', {
    isSelf,
    channelId: session.channelId,
    userId: session.userId,
  })
  const draft = await buildMessageDraft(input)
  const enrichedElements = await enrichAtElements(ctx, session, draft.elements)
  await deliverChatMessageToClients({
    clients: Object.values(ctx.console.clients),
    payload: buildChatPayload(session, draft, enrichedElements),
    roles: input.service.auth.getRoles(),
    getUserRoleIds: (userId) => input.service.auth.getUserRoleIds(userId),
    listBindingsByAuthId: (authId) => ctx.database.get('binding', { aid: authId }),
  })
}

interface BroadcastInput extends BroadcastOptions {
  readonly botLoginInfoCache: Map<string, { userId: string; nickname: string }>
  readonly session: any
  readonly isSelf: boolean
}

async function buildMessageDraft(input: BroadcastInput) {
  const contentAndElements = await resolveMessageContent(input)
  const quotePayload = buildQuotePayload(input.session.quote)
  const elementList = Array.isArray(contentAndElements.elements)
    ? contentAndElements.elements
    : (contentAndElements.content ? h.parse(contentAndElements.content) : [])
  const elements = prependQuoteElement(elementList, quotePayload)
  const content = serializeSpecialElements(contentAndElements.content, elements, input.isSelf)
  input.chatImageAccess.remember(elements, input.session.guildId)
  return {
    content,
    elements,
    guildAvatar: await resolveGuildAvatar(input.ctx, input.session),
    guildName: await resolveGuildName(input.ctx, input.session),
    author: await resolveAuthor(input),
  }
}

async function resolveMessageContent(input: BroadcastInput) {
  let content = input.session.content
  let elements = input.session.elements
  const channelId = input.session.channelId || input.session.guildId
  if (!input.isSelf || !input.session.messageId || !channelId) return { content, elements }
  const bot = input.session.bot || input.ctx.bots.find(b => b.selfId === input.session.selfId)
  if (typeof bot?.getMessage !== 'function') return { content, elements }
  try {
    const message = await bot.getMessage(channelId, input.session.messageId)
    content = message?.content || content
    elements = Array.isArray(message?.elements) ? message.elements : elements
  } catch (e) {
    input.ctx.logger('stuhelperGroupCenter').warn('获取自身消息详情失败: %s', toApiErrorMessage(e))
  }
  return { content, elements }
}

function serializeSpecialElements(content: string, elements: any[], isSelf: boolean) {
  if (isSelf || !Array.isArray(elements) || elements.length === 0) return content
  const hasSpecialElements = elements.some(el =>
    el.type === 'quote' || el.type === 'at' || el.type === 'img' || el.type === 'image' || el.type === 'face')
  return hasSpecialElements ? serializeChatElements(elements) : content
}

async function resolveGuildAvatar(ctx: Context, session: any) {
  const eventAvatar = session.event?.guild?.avatar
  if (eventAvatar) return eventAvatar
  const guild = await resolveGuild(ctx, session)
  return guild?.avatar || (isQQPlatform(session.platform) && session.guildId ? guildAvatar(session.guildId) : undefined)
}

async function resolveGuildName(ctx: Context, session: any) {
  const eventName = session.guildName || session.event?.guild?.name
  if (eventName) return eventName
  return (await resolveGuild(ctx, session))?.name
}

async function resolveGuild(ctx: Context, session: any) {
  if (!session.guildId) return null
  const bot = session.bot || ctx.bots.find(b => b.platform === session.platform)
  if (!bot) return null
  try {
    return await bot.getGuild(session.guildId)
  } catch (e) {
    ctx.logger('stuhelperGroupCenter').warn('获取消息群信息失败: %s', toApiErrorMessage(e))
    return null
  }
}

async function resolveAuthor(input: BroadcastInput) {
  const { session, isSelf } = input
  let username = session.author?.name || session.author?.nick || session.userId
  let avatar = session.author?.avatar
  if (isSelf) {
    const bot = session.bot || input.ctx.bots.find(b => b.selfId === session.selfId)
    const loginInfo = bot ? await getBotLoginInfo(input, bot) : null
    username = loginInfo?.nickname || bot?.user?.name || username || '我'
    avatar = bot?.user?.avatar || avatar
  }
  const targetId = isSelf ? session.selfId : session.userId
  avatar ||= isQQPlatform(session.platform) && targetId ? qqAvatar(targetId) : undefined
  return { username, avatar }
}

async function getBotLoginInfo(input: BroadcastInput, bot: any) {
  const cacheKey = `${bot.platform}:${bot.selfId}`
  const cached = input.botLoginInfoCache.get(cacheKey)
  if (cached) return cached
  const result = await fetchBotLoginInfo(input.ctx, bot)
  if (result) input.botLoginInfoCache.set(cacheKey, result)
  return result
}

async function fetchBotLoginInfo(ctx: Context, bot: any) {
  if (typeof bot.getLogin === 'function') {
    try {
      const login = await bot.getLogin()
      const user = login?.user || bot.user
      if (user?.id || user?.name) {
        return { userId: String(user.id || bot.selfId), nickname: String(user.name || user.nick || user.username || user.id || bot.selfId) }
      }
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').warn('获取 bot 登录信息失败: %s', toApiErrorMessage(e))
    }
  }
  if (bot.user?.name || bot.user?.id) return { userId: bot.selfId, nickname: bot.user.name || bot.selfId }
  return null
}

async function enrichAtElements(ctx: Context, session: any, elements: any[]) {
  return Promise.all(elements.map(async (el: any) => {
    if (el.type !== 'at' || !el.attrs?.id || el.attrs.name) return el
    const bot = session.bot || ctx.bots.find(b => b.platform === session.platform)
    if (!bot || !session.guildId) return el
    try {
      const member = await bot.getGuildMember(session.guildId, el.attrs.id)
      el.attrs.name = member?.nick || member?.user?.name || el.attrs.name
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').warn('获取 at 用户群名片失败: %s', toApiErrorMessage(e))
    }
    return el
  }))
}

function buildChatPayload(session: any, draft: any, enrichedElements: any[]) {
  return {
    id: session.messageId || session.id || Date.now().toString(),
    timestamp: session.timestamp || Date.now(),
    userId: session.userId || session.selfId,
    username: draft.author.username,
    avatar: draft.author.avatar,
    content: draft.content || session.content,
    elements: enrichedElements,
    platform: session.platform,
    guildId: session.guildId,
    guildName: draft.guildName,
    guildAvatar: draft.guildAvatar,
    channelId: session.channelId,
    channelName: session.channelName || session.event?.channel?.name,
    selfId: session.selfId,
  }
}

function buildQuotePayload(quote: any) {
  const messageId = readMessageId(quote)
  if (!messageId) return undefined
  const content = quote.content || readQuoteElementsPreview(quote.elements)
  return { id: messageId, messageId, user: quote.user?.name || quote.user?.id || '', content: content.slice(0, 100) }
}

function readMessageId(message: any) {
  return message?.id || message?.messageId || ''
}

function readQuoteElementsPreview(elements: any[] | undefined): string {
  if (!Array.isArray(elements)) return ''
  return elements.map((element) => {
    if (element?.type === 'text') return element.attrs?.content || element.attrs?.text || ''
    return `[${element?.type || 'unknown'}]`
  }).join('')
}

function isQQPlatform(platform: string) {
  return QQ_PLATFORMS.includes(platform)
}

function qqAvatar(userId: string) {
  return `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
}

function guildAvatar(guildId: string) {
  return `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/`
}
