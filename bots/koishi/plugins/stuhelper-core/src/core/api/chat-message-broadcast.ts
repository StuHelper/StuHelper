import { h } from 'koishi'

import type { WebSocketAPIContext } from './api-context'
import type { ChatImageAccessRegistry } from './chat-image-fetch'
import { deliverChatMessageToClients } from './chat-delivery'
import { prependQuoteElement, serializeChatElements } from './chat-element-serializer'
import { toApiErrorMessage } from './api-response'

interface BroadcastDeps {
  readonly api: WebSocketAPIContext
  readonly imageAccess: ChatImageAccessRegistry
  readonly botLoginInfoCache: Map<string, { userId: string; nickname: string }>
}

interface MessageContent {
  content: string
  elements: any[]
}

interface GuildMeta {
  guildName?: string
  guildAvatar?: string
}

interface AuthorMeta {
  username: string
  avatar?: string
}

const QUOTE_PREVIEW_LENGTH = 100

export function registerChatMessageBroadcast(
  api: WebSocketAPIContext,
  imageAccess: ChatImageAccessRegistry,
): void {
  const deps: BroadcastDeps = { api, imageAccess, botLoginInfoCache: new Map() }
  api.ctx.on('message', (session) => broadcastMessage(deps, session))
  api.ctx.logger('stuhelperGroupCenter').info('Chat message listener registered')
  api.ctx.on('send', (session) => broadcastMessage(deps, session, true))
}

async function broadcastMessage(deps: BroadcastDeps, session: any, isSelf = false) {
  deps.api.ctx.logger('stuhelperGroupCenter').debug('broadcastMessage called:', {
    isSelf,
    channelId: session.channelId,
    userId: session.userId,
  })

  const message = await buildMessageContent(deps, session, isSelf)
  deps.imageAccess.remember(message.elements, session.guildId)
  await sendBroadcastPayload(deps, session, {
    message,
    guild: await readGuildMeta(deps, session),
    author: await readAuthorMeta(deps, session, isSelf),
    enrichedElements: await enrichAtElements(deps, session, message.elements),
  })
}

async function buildMessageContent(deps: BroadcastDeps, session: any, isSelf: boolean): Promise<MessageContent> {
  const contentState = await readSessionMessageContent(deps, session, isSelf)
  const quotePayload = buildQuotePayload(session.quote)
  const elementList = Array.isArray(contentState.elements)
    ? contentState.elements
    : (contentState.content ? h.parse(contentState.content) : [])
  const elementsWithQuote = prependQuoteElement(elementList, quotePayload)
  const finalElements = elementsWithQuote.length > 0
    ? elementsWithQuote
    : readFallbackElements(session, contentState)
  return {
    content: serializeSpecialElements(isSelf, contentState.content, finalElements),
    elements: finalElements,
  }
}

async function readSessionMessageContent(deps: BroadcastDeps, session: any, isSelf: boolean) {
  const state = { content: session.content || '', elements: session.elements }
  const messageChannelId = session.channelId || session.guildId
  if (!isSelf || !session.messageId || !messageChannelId) return state

  const bot = session.bot || deps.api.ctx.bots.find((item) => item.selfId === session.selfId)
  if (typeof bot?.getMessage !== 'function') return state

  try {
    const message = await bot.getMessage(messageChannelId, session.messageId)
    if (message?.content) state.content = message.content
    if (Array.isArray(message?.elements)) state.elements = message.elements
  } catch (cause) {
    deps.api.ctx.logger('stuhelperGroupCenter').warn('获取自身消息详情失败: %s', toApiErrorMessage(cause))
  }
  return state
}

function readFallbackElements(session: any, contentState: { content: string, elements: any }) {
  if (Array.isArray(contentState.elements) && contentState.elements.length > 0) {
    return contentState.elements
  }
  if (Array.isArray(session.elements) && session.elements.length > 0) {
    return session.elements
  }
  return contentState.content ? h.parse(contentState.content) : []
}

function serializeSpecialElements(isSelf: boolean, content: string, elements: any[]) {
  if (isSelf || !Array.isArray(elements) || elements.length === 0) return content
  const hasSpecialElements = elements.some((el) => {
    return el.type === 'quote' || el.type === 'at' || el.type === 'img' || el.type === 'image' || el.type === 'face'
  })
  return hasSpecialElements ? serializeChatElements(elements) : content
}

async function readGuildMeta(deps: BroadcastDeps, session: any): Promise<GuildMeta> {
  const guild = {
    guildName: session.guildName || session.event?.guild?.name,
    guildAvatar: session.event?.guild?.avatar,
  }
  if (session.guildId && (!guild.guildName || !guild.guildAvatar)) {
    await fillGuildMetaFromBot(deps, session, guild)
  }
  if (!guild.guildAvatar && isQQPlatform(session.platform) && session.guildId) {
    guild.guildAvatar = `https://p.qlogo.cn/gh/${session.guildId}/${session.guildId}/640/`
  }
  return guild
}

async function fillGuildMetaFromBot(deps: BroadcastDeps, session: any, guild: GuildMeta) {
  const bot = session.bot || deps.api.ctx.bots.find((item) => item.platform === session.platform)
  if (!bot) return
  try {
    const guildInfo = await bot.getGuild(session.guildId)
    guild.guildName ||= guildInfo?.name
    guild.guildAvatar ||= guildInfo?.avatar
  } catch (cause) {
    deps.api.ctx.logger('stuhelperGroupCenter').warn('获取消息群信息失败: %s', toApiErrorMessage(cause))
  }
}

async function readAuthorMeta(deps: BroadcastDeps, session: any, isSelf: boolean): Promise<AuthorMeta> {
  const author = {
    username: session.author?.name || session.author?.nick || session.userId,
    avatar: session.author?.avatar,
  }
  if (isSelf) {
    await fillSelfAuthorMeta(deps, session, author)
  }
  if (!author.avatar && isQQPlatform(session.platform)) {
    const targetId = isSelf ? session.selfId : session.userId
    if (targetId) author.avatar = `https://q1.qlogo.cn/g?b=qq&nk=${targetId}&s=640`
  }
  return author
}

async function fillSelfAuthorMeta(deps: BroadcastDeps, session: any, author: AuthorMeta) {
  const bot = session.bot || deps.api.ctx.bots.find((item) => item.selfId === session.selfId)
  if (!bot) return
  const loginInfo = await getBotLoginInfo(deps, bot)
  author.username = loginInfo?.nickname || bot.user?.name || author.username || '我'
  author.avatar = bot.user?.avatar || author.avatar
}

async function getBotLoginInfo(deps: BroadcastDeps, bot: any) {
  const cacheKey = `${bot.platform}:${bot.selfId}`
  const cached = deps.botLoginInfoCache.get(cacheKey)
  if (cached) return cached

  const loginInfo = await fetchBotLoginInfo(deps, bot)
  if (loginInfo) deps.botLoginInfoCache.set(cacheKey, loginInfo)
  return loginInfo
}

async function fetchBotLoginInfo(deps: BroadcastDeps, bot: any) {
  if (typeof bot.getLogin === 'function') {
    try {
      const login = await bot.getLogin()
      const user = login?.user || bot.user
      if (user?.id || user?.name) return toBotLoginInfo(user, bot.selfId)
    } catch (cause) {
      deps.api.ctx.logger('stuhelperGroupCenter').warn('获取 bot 登录信息失败: %s', toApiErrorMessage(cause))
    }
  }
  if (bot.user?.name || bot.user?.id) {
    return { userId: bot.selfId, nickname: bot.user.name || bot.selfId }
  }
  return null
}

function toBotLoginInfo(user: any, selfId: string) {
  return {
    userId: String(user.id || selfId),
    nickname: String(user.name || user.nick || user.username || user.id || selfId),
  }
}

async function enrichAtElements(deps: BroadcastDeps, session: any, elements: any[]) {
  return Promise.all(elements.map(async (element) => {
    if (element.type === 'at' && element.attrs?.id && !element.attrs.name) {
      await fillAtElementName(deps, session, element)
    }
    return element
  }))
}

async function fillAtElementName(deps: BroadcastDeps, session: any, element: any) {
  const bot = session.bot || deps.api.ctx.bots.find((item) => item.platform === session.platform)
  if (!bot || !session.guildId) return
  try {
    const member = await bot.getGuildMember(session.guildId, element.attrs.id)
    if (member?.nick || member?.user?.name) {
      element.attrs.name = member.nick || member.user.name
    }
  } catch (cause) {
    deps.api.ctx.logger('stuhelperGroupCenter').warn('获取 at 用户群名片失败: %s', toApiErrorMessage(cause))
  }
}

async function sendBroadcastPayload(deps: BroadcastDeps, session: any, data: any) {
  await deliverChatMessageToClients({
    clients: Object.values(deps.api.ctx.console.clients),
    payload: {
      id: session.messageId || session.id || Date.now().toString(),
      timestamp: session.timestamp || Date.now(),
      userId: session.userId || session.selfId,
      username: data.author.username,
      avatar: data.author.avatar,
      content: data.message.content || session.content,
      elements: data.enrichedElements,
      platform: session.platform,
      guildId: session.guildId,
      guildName: data.guild.guildName,
      guildAvatar: data.guild.guildAvatar,
      channelId: session.channelId,
      channelName: session.channelName || session.event?.channel?.name,
      selfId: session.selfId,
    },
    roles: deps.api.service.auth.getRoles(),
    getUserRoleIds: (userId) => deps.api.service.auth.getUserRoleIds(userId),
    listBindingsByAuthId: (authId) => deps.api.ctx.database.get('binding', { aid: authId }),
  })
}

function buildQuotePayload(quote: any) {
  const messageId = readMessageId(quote)
  if (!messageId) return undefined

  const content = quote.content || readQuoteElementsPreview(quote.elements)
  return {
    id: messageId,
    messageId,
    user: quote.user?.name || quote.user?.id || '',
    content: content.slice(0, QUOTE_PREVIEW_LENGTH),
  }
}

function readMessageId(message: any) {
  return message?.id || message?.messageId || ''
}

function readQuoteElementsPreview(elements: any[] | undefined) {
  if (!Array.isArray(elements)) return ''
  return elements.map((element) => {
    if (element?.type === 'text') {
      return element.attrs?.content || element.attrs?.text || ''
    }
    return `[${element?.type || 'unknown'}]`
  }).join('')
}

function isQQPlatform(platform: string | undefined) {
  return platform === 'onebot' || platform === 'red' || platform === 'qq'
}
