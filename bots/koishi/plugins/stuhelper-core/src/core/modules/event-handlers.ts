import type { GroupConfig } from '../../types'
import {
  DEFAULT_LEVEL_LIMIT,
  DEFAULT_MEMBER_REQUEST_CONFIG,
  botInternal,
  eventLogger,
  groupConfigOf,
  matchesAnyKeyword,
  requestCommentOf,
  requestIdOf,
  type EventRuntimeHost,
  type EventSession,
  type GroupRequest,
} from './event-support'
import {
  handleGuildMemberAdded,
  handleGuildMemberRemoved,
  setupEventScheduledTasks,
} from './event-member-handlers'

export { setupEventScheduledTasks }

export function registerEventListeners(host: EventRuntimeHost): void {
  host.ctx.on('friend-request', async session => {
    await handleFriendRequest(host, session as EventSession)
  })
  host.ctx.on('guild-request', async session => {
    await handleGuildRequest(host, session as EventSession)
  })
  host.ctx.on('guild-member-request', async session => {
    await handleGuildMemberRequest(host, session as EventSession)
  })
  host.ctx.on('guild-member-added', async session => {
    await handleGuildMemberAdded(host, session as EventSession)
  })
  host.ctx.on('guild-member-removed', async session => {
    await handleGuildMemberRemoved(host, session as EventSession)
  })
}

async function handleFriendRequest(host: EventRuntimeHost, session: EventSession) {
  if (await rejectBlacklistedFriendRequest(host, session)) return

  const config = host.config.friendRequest
  if (!config?.enabled) return
  const comment = requestCommentOf(session)
  if (!comment || !config.keywords?.length) return

  if (matchesAnyKeyword(comment, config.keywords)) {
    await session.bot.handleFriendRequest(requestIdOf(session), true)
    return
  }

  await session.bot.handleFriendRequest(requestIdOf(session), false, config.rejectMessage || '验证信息不正确')
}

async function handleGuildRequest(host: EventRuntimeHost, session: EventSession) {
  if (await rejectBlacklistedGroupRequest(host, {
    session,
    failureLog: '拒绝群邀请失败:',
  })) return

  if (host.config.guildRequest?.enabled) {
    await session.bot.handleGuildRequest(requestIdOf(session), true)
    return
  }

  await session.bot.handleGuildRequest(
    requestIdOf(session),
    false,
    host.config.guildRequest?.rejectMessage || '暂不接受群邀请',
  )
}

async function handleGuildMemberRequest(host: EventRuntimeHost, session: EventSession) {
  if (await rejectBlacklistedGroupRequest(host, {
    session,
    failureLog: '拒绝入群申请失败:',
  })) return
  if (await rejectDuringLeaveCooldown(host, session)) return

  const groupConfig = groupConfigOf(host, session.guildId, DEFAULT_MEMBER_REQUEST_CONFIG)
  if (await rejectBelowLevelLimit(session, groupConfig)) return

  await acceptIfKeywordMatches(host, session, groupConfig)
}

async function rejectBlacklistedFriendRequest(
  host: EventRuntimeHost,
  session: EventSession,
): Promise<boolean> {
  if (!host.data.blacklist.getAll()[session.userId]) return false

  try {
    await session.bot.handleFriendRequest(requestIdOf(session), false, '您在黑名单中')
  } catch (error) {
    eventLogger.error('拒绝好友请求失败:', error)
  }
  return true
}

async function rejectBlacklistedGroupRequest(
  host: EventRuntimeHost,
  request: GroupRequest,
): Promise<boolean> {
  if (!host.data.blacklist.getAll()[request.session.userId]) return false

  try {
    await rejectGuildOrMemberRequest(request.session, '您在黑名单中')
  } catch (error) {
    eventLogger.error(request.failureLog, error)
  }
  return true
}

async function rejectDuringLeaveCooldown(
  host: EventRuntimeHost,
  session: EventSession,
): Promise<boolean> {
  const record = host.data.leaveRecords.getAll()[`${session.guildId}_${session.userId}`]
  if (!record || Date.now() >= record.expireTime) return false

  try {
    await session.bot.handleGuildMemberRequest(requestIdOf(session), false, '退群后需要等待冷却时间才能重新加入')
  } catch (error) {
    eventLogger.error('拒绝入群申请失败:', error)
  }
  return true
}

async function rejectBelowLevelLimit(
  session: EventSession,
  groupConfig: GroupConfig,
): Promise<boolean> {
  const levelLimit = groupConfig.levelLimit ?? DEFAULT_LEVEL_LIMIT
  if (levelLimit <= DEFAULT_LEVEL_LIMIT) return false

  try {
    const userInfo = await botInternal(session).getStrangerInfo(session.userId, true)
    if (userInfo.level >= levelLimit) return false

    await session.bot.handleGuildMemberRequest(requestIdOf(session), false, `等级不足${levelLimit}级`)
    return true
  } catch (error) {
    eventLogger.error('获取用户信息失败:', error)
    return false
  }
}

async function acceptIfKeywordMatches(
  host: EventRuntimeHost,
  session: EventSession,
  groupConfig: GroupConfig,
) {
  const approvalKeywords = Array.isArray(groupConfig.approvalKeywords)
    ? groupConfig.approvalKeywords
    : []
  const keywords = [...(host.config.keywords || []), ...approvalKeywords]
  const comment = requestCommentOf(session)

  if (!comment || !matchesAnyKeyword(comment, keywords)) return
  await session.bot.handleGuildMemberRequest(requestIdOf(session), true)
}

async function rejectGuildOrMemberRequest(session: EventSession, message: string): Promise<void> {
  if (session.type === 'guild-request') {
    await session.bot.handleGuildRequest(requestIdOf(session), false, message)
    return
  }
  await session.bot.handleGuildMemberRequest(requestIdOf(session), false, message)
}
