import type { Bot } from 'koishi'

import type { ChatImageFetchParams, ChatImageAccessRegistry } from './chat-image-fetch'
import { fetchOneBotImage } from './chat-image-fetch'
import type { WebSocketAPIContext } from './api-context'
import { error, success, toApiErrorMessage } from './api-response'
import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'
import { registerChatMessageBroadcast } from './chat-message-broadcast'
import {
  isGuildAdminMember,
  isGuildOwnerMember,
  type GuildAdminMember,
} from '../services/auth-guild-admin'

const QQ_PLATFORMS = new Set(['onebot', 'red', 'qq'])
const MAX_CHAT_CONTENT_BYTES = 256 * 1024

export interface ChatAPIOptions {
  readonly imageAccess: ChatImageAccessRegistry
}

export function registerChatAPI(api: WebSocketAPIContext, options: ChatAPIOptions): void {
  api.addAuthorityListener('stuhelperGroupCenter/chat/guild-members', async function (params: { guildId: string }) {
    return handleGuildMembers(api, this, params.guildId)
  })
  api.addAuthorityListener('stuhelperGroupCenter/chat/guild-info', async function (params: { guildId: string }) {
    return handleGuildInfo(api, this, params.guildId)
  })
  api.addAuthorityListener('stuhelperGroupCenter/chat/user-info', async function (params: { userId: string }) {
    return handleUserInfo(api, this, params.userId)
  })
  api.addAuthorityListener('stuhelperGroupCenter/chat/send', async function (params: ChatSendParams) {
    return handleChatSend(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/chat/recall', async function (params: ChatRecallParams) {
    return handleChatRecall(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/image/fetch', async function (params: ChatImageFetchParams) {
    return handleImageFetch({ api, imageAccess: options.imageAccess, client: this, params })
  })
  registerChatMessageBroadcast(api, options.imageAccess)
}

interface ChatSendParams {
  readonly channelId: string
  readonly content: string
  readonly platform?: string
  readonly guildId?: string
}

interface ChatRecallParams {
  readonly channelId: string
  readonly messageId: string
  readonly platform?: string
  readonly guildId?: string
}

interface ChatGuildMemberProfile {
  readonly id: string
  readonly name: string
  readonly avatar: string
  readonly isAdmin: boolean
  readonly isOwner: boolean
  readonly title: string
  readonly joinedAt?: number | string | Date
}

type GuildMemberListBot = Pick<Bot, 'platform' | 'getGuildMemberList'>
type ChatGuildMember = GuildAdminMember & {
  readonly userId?: string
  readonly name?: string
  readonly avatar?: string
  readonly title?: string
  readonly joinedAt?: unknown
}

async function handleGuildMembers(api: WebSocketAPIContext, client: unknown, guildId: string) {
  try {
    if (!guildId) return error('缺少 guildId 参数')
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, guildId, 'chat guild members')
    return await fetchFormattedGuildMembers(api, guildId)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取群成员列表失败')
  }
}

async function fetchFormattedGuildMembers(api: WebSocketAPIContext, guildId: string) {
  for (const bot of api.ctx.bots) {
    api.ctx.logger('stuhelperGroupCenter').debug('Trying bot:', bot.platform, bot.selfId)
    try {
      const members = (await fetchAllGuildMembers(bot, guildId)).map((member) => formatGuildMember(bot, member))
      members.sort(compareGuildMembers)
      return success({ members, total: members.length })
    } catch (cause) {
      api.ctx.logger('stuhelperGroupCenter').warn('获取群成员列表失败:', cause)
    }
  }
  return error('无法获取群成员列表')
}

async function handleGuildInfo(api: WebSocketAPIContext, client: unknown, guildId: string) {
  try {
    if (!guildId) return error('缺少 guildId 参数')
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, guildId, 'chat guild info')
    return await fetchGuildInfo(api, guildId)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取群信息失败')
  }
}

async function fetchGuildInfo(api: WebSocketAPIContext, guildId: string) {
  const failures: string[] = []
  for (const bot of api.ctx.bots) {
    try {
      const guild = await bot.getGuild(guildId)
      if (guild) return success({ name: guild.name, avatar: readGuildAvatar(bot, guildId, guild.avatar) })
    } catch (cause) {
      const message = toApiErrorMessage(cause)
      failures.push(message)
      api.ctx.logger('stuhelperGroupCenter').warn('获取群信息失败: %s', message)
    }
  }
  return error(`无法获取群信息${lastFailureSuffix(failures)}`)
}

async function handleUserInfo(api: WebSocketAPIContext, client: unknown, userId: string) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'chat user info')
    if (!userId) return error('缺少 userId 参数')
    return await fetchUserInfo(api, userId)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取用户信息失败')
  }
}

async function fetchUserInfo(api: WebSocketAPIContext, userId: string) {
  const failures: string[] = []
  for (const bot of api.ctx.bots) {
    try {
      const user = await bot.getUser(userId)
      if (user) return success({ name: user.name || user.nick || userId, avatar: readUserAvatar(bot, userId, user.avatar) })
    } catch (cause) {
      const message = toApiErrorMessage(cause)
      failures.push(message)
      api.ctx.logger('stuhelperGroupCenter').warn('获取用户信息失败: %s', message)
    }
  }
  return error(`无法获取用户信息${lastFailureSuffix(failures)}`)
}

async function handleChatSend(api: WebSocketAPIContext, client: unknown, params: ChatSendParams) {
  try {
    if (!params.channelId || !params.content) return error('缺少必要参数')
    assertChatContentSize(params.content)
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, params.guildId, 'chat send')
    const bot = api.ctx.bots.find((item) => !params.platform || item.platform === params.platform)
    if (!bot) return error('未找到可用的 Bot')
    await bot.sendMessage(params.channelId, params.content, params.guildId)
    return success({ success: true })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('发送消息失败:', cause)
    return error(cause instanceof Error ? cause.message : '发送失败')
  }
}

function assertChatContentSize(content: string) {
  if (Buffer.byteLength(content, 'utf8') > MAX_CHAT_CONTENT_BYTES) {
    throw new Error(`message content is too large; max ${MAX_CHAT_CONTENT_BYTES} bytes`)
  }
}

async function handleChatRecall(api: WebSocketAPIContext, client: unknown, params: ChatRecallParams) {
  try {
    if (!params.channelId || !params.messageId) return error('缺少必要参数')
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, params.guildId, 'chat recall')
    const bot = api.ctx.bots.find((item) => !params.platform || item.platform === params.platform)
    if (!bot) return error('未找到可用的 Bot')
    await bot.deleteMessage(params.channelId, params.messageId)
    return success({ success: true })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('撤回消息失败:', cause)
    return error(cause instanceof Error ? cause.message : '撤回失败')
  }
}

async function handleImageFetch(input: {
  readonly api: WebSocketAPIContext
  readonly imageAccess: ChatImageAccessRegistry
  readonly client: unknown
  readonly params: ChatImageFetchParams
}) {
  const { api, imageAccess, client, params } = input
  try {
    const scope = await api.resolveConsoleScope(client)
    const request = imageAccess.assertAllowed(params, scope)
    const result = await fetchOneBotImage(api.ctx.bots, request, api.ctx.logger('stuhelperGroupCenter'))
    return success(result)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取图片失败')
  }
}

async function fetchAllGuildMembers(bot: GuildMemberListBot, guildId: string): Promise<ChatGuildMember[]> {
  const members: ChatGuildMember[] = []
  let next: string | undefined
  do {
    const result = await bot.getGuildMemberList(guildId, next)
    if (result.data) members.push(...result.data)
    next = result.next
  } while (next)
  return members
}

function formatGuildMember(bot: Pick<Bot, 'platform'>, member: ChatGuildMember): ChatGuildMemberProfile {
  const userId = member.user?.id || member.userId || ''
  return {
    id: userId,
    name: member.nick || member.user?.nick || member.user?.name || member.name || userId,
    avatar: readUserAvatar(bot, userId, member.user?.avatar || member.avatar),
    isAdmin: isGuildAdminMember(member),
    isOwner: isGuildOwnerMember(member),
    title: member.title || '',
    joinedAt: readJoinedAt(member.joinedAt),
  }
}

function compareGuildMembers(a: ChatGuildMemberProfile, b: ChatGuildMemberProfile) {
  if (a.isOwner && !b.isOwner) return -1
  if (!a.isOwner && b.isOwner) return 1
  if (a.isAdmin && !b.isAdmin) return -1
  if (!a.isAdmin && b.isAdmin) return 1
  return (a.name || '').localeCompare(b.name || '')
}

function readGuildAvatar(bot: Pick<Bot, 'platform'>, guildId: string, avatar?: string) {
  if (avatar) return avatar
  if (!QQ_PLATFORMS.has(bot.platform)) return ''
  return `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/`
}

function readUserAvatar(bot: Pick<Bot, 'platform'>, userId: string, avatar?: string) {
  if (avatar) return avatar
  if (!userId || !QQ_PLATFORMS.has(bot.platform)) return ''
  return `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
}

function readJoinedAt(value: unknown) {
  if (typeof value === 'number' || typeof value === 'string' || value instanceof Date) return value
  return undefined
}

function lastFailureSuffix(failures: readonly string[]) {
  return failures.length > 0 ? `: ${failures[failures.length - 1]}` : ''
}
