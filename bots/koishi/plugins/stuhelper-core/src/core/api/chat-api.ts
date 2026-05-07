import {
  createChatImageAccessRegistry,
  fetchOneBotImage,
  type ChatImageFetchParams,
} from './chat-image-fetch'
import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'
import { error, success, toApiErrorMessage } from './api-response'
import { registerChatMessageBroadcast } from './chat-message-broadcast'
import type { WebSocketAPIContext } from './websocket-api-context'

const MAX_CHAT_CONTENT_BYTES = 256 * 1024
const QQ_PLATFORMS = ['onebot', 'red', 'qq']

export function registerChatAPI(api: WebSocketAPIContext) {
  const { ctx, addAuthorityListener, resolveConsoleScope } = api
  const chatImageAccess = createChatImageAccessRegistry()

  addAuthorityListener('stuhelperGroupCenter/chat/guild-members', async function (params: { guildId: string }) {
    try {
      if (!params.guildId) return error('缺少 guildId 参数')
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'chat guild members')
      return await listGuildMembers(api, params.guildId)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取群成员列表失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/chat/guild-info', async function (params: { guildId: string }) {
    try {
      if (!params.guildId) return error('缺少 guildId 参数')
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'chat guild info')
      return await getGuildInfo(api, params.guildId)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取群信息失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/chat/user-info', async function (params: { userId: string }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'chat user info')
      if (!params.userId) return error('缺少 userId 参数')
      return await getUserInfo(api, params.userId)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取用户信息失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/chat/send', async function (params: { channelId: string, content: string, platform?: string, guildId?: string }) {
    try {
      if (!params.channelId || !params.content) return error('缺少必要参数')
      assertChatContentSize(params.content)
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'chat send')
      const bot = ctx.bots.find(b => !params.platform || b.platform === params.platform)
      if (!bot) return error('未找到可用的 Bot')
      await bot.sendMessage(params.channelId, params.content, params.guildId)
      return success({ success: true })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('发送消息失败:', e)
      return error(e instanceof Error ? e.message : '发送失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/chat/recall', async function (params: { channelId: string, messageId: string, platform?: string, guildId?: string }) {
    try {
      if (!params.channelId || !params.messageId) return error('缺少必要参数')
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'chat recall')
      const bot = ctx.bots.find(b => !params.platform || b.platform === params.platform)
      if (!bot) return error('未找到可用的 Bot')
      await bot.deleteMessage(params.channelId, params.messageId)
      return success({ success: true })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('撤回消息失败:', e)
      return error(e instanceof Error ? e.message : '撤回失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/image/fetch', async function (params: ChatImageFetchParams) {
    try {
      const scope = await resolveConsoleScope(this)
      const request = chatImageAccess.assertAllowed(params, scope)
      const result = await fetchOneBotImage(ctx.bots, request, ctx.logger('stuhelperGroupCenter'))
      return success(result)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取图片失败')
    }
  })

  registerChatMessageBroadcast({ ctx, service: api.service, chatImageAccess })
}

async function listGuildMembers(api: WebSocketAPIContext, guildId: string) {
  const { ctx } = api
  ctx.logger('stuhelperGroupCenter').debug('getGuildMembers called:', guildId)
  for (const bot of ctx.bots) {
    try {
      ctx.logger('stuhelperGroupCenter').debug('Trying bot:', bot.platform, bot.selfId)
      const members = await fetchGuildMembers(bot, guildId)
      const formatted = formatGuildMembers(members, bot.platform)
      return success({ members: formatted, total: formatted.length })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').warn('获取群成员列表失败:', e)
    }
  }
  return error('无法获取群成员列表')
}

async function getGuildInfo(api: WebSocketAPIContext, guildId: string) {
  const failures: string[] = []
  for (const bot of api.ctx.bots) {
    try {
      const guild = await bot.getGuild(guildId)
      if (guild) return success({ name: guild.name, avatar: guild.avatar || fallbackGuildAvatar(bot.platform, guildId) })
    } catch (e) {
      failures.push(toApiErrorMessage(e))
      api.ctx.logger('stuhelperGroupCenter').warn('获取群信息失败: %s', failures[failures.length - 1])
    }
  }
  return error(`无法获取群信息${lastFailureSuffix(failures)}`)
}

async function getUserInfo(api: WebSocketAPIContext, userId: string) {
  const failures: string[] = []
  for (const bot of api.ctx.bots) {
    try {
      const user = await bot.getUser(userId)
      if (user) return success({ name: user.name || user.nick || userId, avatar: user.avatar || fallbackQQAvatar(bot.platform, userId) })
    } catch (e) {
      failures.push(toApiErrorMessage(e))
      api.ctx.logger('stuhelperGroupCenter').warn('获取用户信息失败: %s', failures[failures.length - 1])
    }
  }
  return error(`无法获取用户信息${lastFailureSuffix(failures)}`)
}

async function fetchGuildMembers(bot: any, guildId: string) {
  const members: any[] = []
  let next: string | undefined
  do {
    const result = await bot.getGuildMemberList(guildId, next)
    if (result.data) members.push(...result.data)
    next = result.next
  } while (next)
  return members
}

function formatGuildMembers(members: any[], platform: string) {
  return members.map(member => {
    const userId = member.user?.id || member.userId
    return {
      id: userId,
      name: member.nick || member.user?.nick || member.user?.name || userId,
      avatar: member.user?.avatar || member.avatar || fallbackQQAvatar(platform, userId),
      isAdmin: member.roles?.includes('admin') || false,
      isOwner: member.roles?.includes('owner') || false,
      title: member.title || '',
      joinedAt: member.joinedAt,
    }
  }).sort(compareGuildMembers)
}

function compareGuildMembers(a: any, b: any) {
  if (a.isOwner && !b.isOwner) return -1
  if (!a.isOwner && b.isOwner) return 1
  if (a.isAdmin && !b.isAdmin) return -1
  if (!a.isAdmin && b.isAdmin) return 1
  return (a.name || '').localeCompare(b.name || '')
}

function assertChatContentSize(content: string) {
  if (Buffer.byteLength(content, 'utf8') > MAX_CHAT_CONTENT_BYTES) {
    throw new Error(`message content is too large; max ${MAX_CHAT_CONTENT_BYTES} bytes`)
  }
}

function lastFailureSuffix(failures: readonly string[]) {
  return failures.length > 0 ? `: ${failures[failures.length - 1]}` : ''
}

function fallbackQQAvatar(platform: string, userId: string) {
  return QQ_PLATFORMS.includes(platform) ? `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640` : undefined
}

function fallbackGuildAvatar(platform: string, guildId: string) {
  return QQ_PLATFORMS.includes(platform) ? `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/` : undefined
}
