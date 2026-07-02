import { h, type Bot } from 'koishi'

import type { WebSocketAPIContext } from './api-context'
import type { ChatImageAccessRegistry } from './chat-image-fetch'
import { deliverChatMessageToClients } from './chat-delivery'
import { prependQuoteElement, serializeChatElements, type ChatElement } from './chat-element-serializer'
import { toApiErrorMessage } from './api-response'

interface ChatUserProfile {
  readonly id?: unknown
  readonly name?: unknown
  readonly nick?: unknown
  readonly username?: unknown
  readonly avatar?: string
}

interface ChatGuildInfo {
  readonly name?: string
  readonly avatar?: string
}

interface ChatChannelInfo {
  readonly name?: string
}

interface ChatQuote {
  readonly id?: unknown
  readonly messageId?: unknown
  readonly content?: unknown
  readonly elements?: unknown
  readonly user?: ChatUserProfile
}

interface ChatBroadcastBot extends Pick<Bot, 'platform' | 'selfId'> {
  readonly user?: ChatUserProfile
  getMessage?(channelId: string, messageId: string): Promise<{ content?: string; elements?: unknown }>
  getGuild?(guildId: string): Promise<ChatGuildInfo | undefined>
  getGuildMember?(guildId: string, userId: string): Promise<{
    nick?: string
    user?: Pick<ChatUserProfile, 'name'>
  } | undefined>
  getLogin?(): Promise<{ user?: ChatUserProfile } | undefined>
}

interface ChatBroadcastSession {
  readonly id?: string | number
  readonly platform?: string
  readonly selfId?: string
  readonly channelId?: string
  readonly channelName?: string
  readonly guildId?: string
  readonly guildName?: string
  readonly userId?: string
  readonly messageId?: string
  readonly timestamp?: number
  readonly content?: string
  readonly elements?: unknown
  readonly quote?: ChatQuote
  readonly author?: {
    readonly name?: string
    readonly nick?: string
    readonly avatar?: string
  }
  readonly event?: {
    readonly guild?: ChatGuildInfo
    readonly channel?: ChatChannelInfo
  }
  readonly bot?: ChatBroadcastBot
}

interface BotLoginInfo {
  readonly userId: string
  readonly nickname: string
}

interface BroadcastDeps {
  readonly api: WebSocketAPIContext
  readonly imageAccess: ChatImageAccessRegistry
  readonly botLoginInfoCache: Map<string, BotLoginInfo>
}

interface MessageContent {
  content: string
  elements: ChatElement[]
}

interface GuildMeta {
  guildName?: string
  guildAvatar?: string
}

interface AuthorMeta {
  username: string
  avatar?: string
}

interface BroadcastPayloadData {
  readonly message: MessageContent
  readonly guild: GuildMeta
  readonly author: AuthorMeta
  readonly enrichedElements: ChatElement[]
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

async function broadcastMessage(deps: BroadcastDeps, session: ChatBroadcastSession, isSelf = false) {
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

async function buildMessageContent(
  deps: BroadcastDeps,
  session: ChatBroadcastSession,
  isSelf: boolean,
): Promise<MessageContent> {
  const contentState = await readSessionMessageContent(deps, session, isSelf)
  const quotePayload = buildQuotePayload(session.quote)
  const elementList = Array.isArray(contentState.elements)
    ? contentState.elements
    : parseContentElements(contentState.content)
  const elementsWithQuote = prependQuoteElement(elementList, quotePayload)
  const finalElements = elementsWithQuote.length > 0
    ? elementsWithQuote
    : readFallbackElements(session, contentState)
  return {
    content: serializeSpecialElements(isSelf, contentState.content, finalElements),
    elements: finalElements,
  }
}

async function readSessionMessageContent(deps: BroadcastDeps, session: ChatBroadcastSession, isSelf: boolean) {
  const state = {
    content: session.content || '',
    elements: readChatElements(session.elements),
  }
  const messageChannelId = session.channelId || session.guildId
  if (!isSelf || !session.messageId || !messageChannelId) return state

  const bot = session.bot || deps.api.ctx.bots.find((item) => item.selfId === session.selfId)
  if (typeof bot?.getMessage !== 'function') return state

  try {
    const message = await bot.getMessage(messageChannelId, session.messageId)
    if (message?.content) state.content = message.content
    const messageElements = readChatElements(message?.elements)
    if (messageElements) state.elements = messageElements
  } catch (cause) {
    deps.api.ctx.logger('stuhelperGroupCenter').warn('获取自身消息详情失败: %s', toApiErrorMessage(cause))
  }
  return state
}

function readFallbackElements(session: ChatBroadcastSession, contentState: { content: string; elements?: ChatElement[] }) {
  if (contentState.elements && contentState.elements.length > 0) {
    return contentState.elements
  }
  const sessionElements = readChatElements(session.elements)
  if (sessionElements && sessionElements.length > 0) {
    return sessionElements
  }
  return parseContentElements(contentState.content)
}

function serializeSpecialElements(isSelf: boolean, content: string, elements: readonly ChatElement[]) {
  if (isSelf || !Array.isArray(elements) || elements.length === 0) return content
  const hasSpecialElements = elements.some((el) => {
    return el.type === 'quote' || el.type === 'at' || el.type === 'img' || el.type === 'image' || el.type === 'face'
  })
  return hasSpecialElements ? serializeChatElements(elements) : content
}

async function readGuildMeta(deps: BroadcastDeps, session: ChatBroadcastSession): Promise<GuildMeta> {
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

async function fillGuildMetaFromBot(deps: BroadcastDeps, session: ChatBroadcastSession, guild: GuildMeta) {
  if (!session.guildId) return
  try {
    // 复用 CacheService 的 TTL 缓存，避免每条消息都向 bot 远程查询群信息。
    const guildInfo = await deps.api.service.cache.getGuildInfo(session.guildId)
    guild.guildName ||= guildInfo?.name
    guild.guildAvatar ||= guildInfo?.avatar
  } catch (cause) {
    deps.api.ctx.logger('stuhelperGroupCenter').warn('获取消息群信息失败: %s', toApiErrorMessage(cause))
  }
}

async function readAuthorMeta(deps: BroadcastDeps, session: ChatBroadcastSession, isSelf: boolean): Promise<AuthorMeta> {
  const author = {
    username: session.author?.name || session.author?.nick || session.userId || '',
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

async function fillSelfAuthorMeta(deps: BroadcastDeps, session: ChatBroadcastSession, author: AuthorMeta) {
  const bot = session.bot || deps.api.ctx.bots.find((item) => item.selfId === session.selfId)
  if (!bot) return
  const loginInfo = await getBotLoginInfo(deps, bot)
  author.username = loginInfo?.nickname || readOptionalString(bot.user?.name) || author.username || '我'
  author.avatar = bot.user?.avatar || author.avatar
}

async function getBotLoginInfo(deps: BroadcastDeps, bot: ChatBroadcastBot) {
  const cacheKey = `${bot.platform}:${bot.selfId}`
  const cached = deps.botLoginInfoCache.get(cacheKey)
  if (cached) return cached

  const loginInfo = await fetchBotLoginInfo(deps, bot)
  if (loginInfo) deps.botLoginInfoCache.set(cacheKey, loginInfo)
  return loginInfo
}

async function fetchBotLoginInfo(deps: BroadcastDeps, bot: ChatBroadcastBot): Promise<BotLoginInfo | null> {
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
    return { userId: bot.selfId, nickname: readOptionalString(bot.user.name) || bot.selfId }
  }
  return null
}

function toBotLoginInfo(user: ChatUserProfile, selfId: string): BotLoginInfo {
  return {
    userId: String(user.id || selfId),
    nickname: String(user.name || user.nick || user.username || user.id || selfId),
  }
}

async function enrichAtElements(
  deps: BroadcastDeps,
  session: ChatBroadcastSession,
  elements: readonly ChatElement[],
): Promise<ChatElement[]> {
  return Promise.all(elements.map(async (element) => {
    if (element.type === 'at' && element.attrs?.id && !element.attrs.name) {
      const enriched = { ...element, attrs: { ...element.attrs } }
      await fillAtElementName(deps, session, enriched)
      return enriched
    }
    return element
  }))
}

async function fillAtElementName(deps: BroadcastDeps, session: ChatBroadcastSession, element: ChatElement) {
  const userId = readOptionalString(element.attrs?.id)
  if (!session.guildId || !userId) return
  try {
    // 复用 CacheService 的 TTL 缓存，避免每个 at 元素都向 bot 远程查询群名片。
    const member = await deps.api.service.cache.getMemberInfo(session.guildId, userId)
    if (member?.nick || member?.name) {
      element.attrs.name = member.nick || member.name
    }
  } catch (cause) {
    deps.api.ctx.logger('stuhelperGroupCenter').warn('获取 at 用户群名片失败: %s', toApiErrorMessage(cause))
  }
}

async function sendBroadcastPayload(
  deps: BroadcastDeps,
  session: ChatBroadcastSession,
  data: BroadcastPayloadData,
) {
  await deliverChatMessageToClients({
    clients: Object.values(deps.api.ctx.console.clients),
    payload: {
      id: readPayloadId(session),
      timestamp: session.timestamp || Date.now(),
      userId: session.userId || session.selfId || '',
      username: data.author.username,
      avatar: data.author.avatar,
      content: data.message.content || '',
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

function readPayloadId(session: ChatBroadcastSession) {
  return readOptionalString(session.messageId) || readOptionalString(session.id) || Date.now().toString()
}

function buildQuotePayload(quote: ChatQuote | undefined) {
  const messageId = readMessageId(quote)
  if (!messageId) return undefined

  const content = readOptionalString(quote?.content) || readQuoteElementsPreview(quote?.elements)
  return {
    id: messageId,
    messageId,
    user: readOptionalString(quote?.user?.name) || readOptionalString(quote?.user?.id) || '',
    content: content.slice(0, QUOTE_PREVIEW_LENGTH),
  }
}

function readMessageId(message: ChatQuote | undefined) {
  return readOptionalString(message?.id) || readOptionalString(message?.messageId) || ''
}

function readQuoteElementsPreview(elements: unknown) {
  if (!Array.isArray(elements)) return ''
  return elements.map((element) => {
    if (!isRecord(element)) return '[unknown]'
    const type = readOptionalString(element.type) || 'unknown'
    if (type === 'text' && isRecord(element.attrs)) {
      return readOptionalString(element.attrs.content) || readOptionalString(element.attrs.text) || ''
    }
    return `[${type}]`
  }).join('')
}

function readChatElements(value: unknown): ChatElement[] | undefined {
  return Array.isArray(value) ? value.filter(isChatElement) : undefined
}

function parseContentElements(content: string): ChatElement[] {
  return content ? h.parse(content).filter(isChatElement) : []
}

function isChatElement(value: unknown): value is ChatElement {
  if (!isRecord(value)) return false
  if (value.type !== undefined && typeof value.type !== 'string') return false
  if (value.attrs !== undefined && !isRecord(value.attrs)) return false
  if (value.children !== undefined && (!Array.isArray(value.children) || !value.children.every(isChatElement))) {
    return false
  }
  return true
}

function readOptionalString(value: unknown): string {
  return typeof value === 'string' || typeof value === 'number' ? String(value) : ''
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isQQPlatform(platform: string | undefined) {
  return platform === 'onebot' || platform === 'red' || platform === 'qq'
}
