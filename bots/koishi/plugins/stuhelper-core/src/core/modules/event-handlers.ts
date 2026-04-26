import type { GroupConfig } from '../../types'
import {
  DEFAULT_LEVEL_LIMIT,
  DEFAULT_MEMBER_REQUEST_CONFIG,
  botInternal,
  eventLogger,
  groupConfigOf,
  matchesAnyKeyword,
  requestDataOf,
  type EventRuntimeHost,
  type EventSession,
  type GroupRequest,
  type RequestData,
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
  const data = requestDataOf(session)
  if (await rejectBlacklistedFriendRequest(host, session, data)) return

  const config = host.config.friendRequest
  if (!config?.enabled) return
  if (!data.comment || !config.keywords?.length) return

  if (matchesAnyKeyword(data.comment, config.keywords)) {
    await botInternal(session).setFriendAddRequest(data.flag, true)
    return
  }

  await botInternal(session).setFriendAddRequest(
    data.flag,
    false,
    config.rejectMessage || '验证信息不正确',
  )
}

async function handleGuildRequest(host: EventRuntimeHost, session: EventSession) {
  const data = requestDataOf(session)
  if (await rejectBlacklistedGroupRequest(host, {
    session,
    data,
    failureLog: '拒绝群邀请失败:',
  })) return

  if (host.config.guildRequest?.enabled) {
    await botInternal(session).setGroupAddRequest(data.flag, data.sub_type, true)
    return
  }

  await botInternal(session).setGroupAddRequest(
    data.flag,
    data.sub_type,
    false,
    host.config.guildRequest?.rejectMessage || '暂不接受群邀请',
  )
}

async function handleGuildMemberRequest(host: EventRuntimeHost, session: EventSession) {
  const data = requestDataOf(session)
  if (await rejectBlacklistedGroupRequest(host, {
    session,
    data,
    failureLog: '拒绝入群申请失败:',
  })) return
  if (await rejectDuringLeaveCooldown(host, session, data)) return

  const groupConfig = groupConfigOf(host, session.guildId, DEFAULT_MEMBER_REQUEST_CONFIG)
  if (await rejectBelowLevelLimit(session, data, groupConfig)) return

  await acceptIfKeywordMatches(host, session, data, groupConfig)
}

async function rejectBlacklistedFriendRequest(
  host: EventRuntimeHost,
  session: EventSession,
  data: RequestData,
): Promise<boolean> {
  if (!host.data.blacklist.getAll()[session.userId]) return false

  try {
    await botInternal(session).setFriendAddRequest(data.flag, false, '您在黑名单中')
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
    await botInternal(request.session).setGroupAddRequest(
      request.data.flag,
      request.data.sub_type,
      false,
      '您在黑名单中',
    )
  } catch (error) {
    eventLogger.error(request.failureLog, error)
  }
  return true
}

async function rejectDuringLeaveCooldown(
  host: EventRuntimeHost,
  session: EventSession,
  data: RequestData,
): Promise<boolean> {
  const record = host.data.leaveRecords.getAll()[`${session.guildId}_${session.userId}`]
  if (!record || Date.now() >= record.expireTime) return false

  try {
    await botInternal(session).setGroupAddRequest(
      data.flag,
      data.sub_type,
      false,
      '退群后需要等待冷却时间才能重新加入',
    )
  } catch (error) {
    eventLogger.error('拒绝入群申请失败:', error)
  }
  return true
}

async function rejectBelowLevelLimit(
  session: EventSession,
  data: RequestData,
  groupConfig: GroupConfig,
): Promise<boolean> {
  const levelLimit = groupConfig.levelLimit ?? DEFAULT_LEVEL_LIMIT
  if (levelLimit <= DEFAULT_LEVEL_LIMIT) return false

  try {
    const userInfo = await botInternal(session).getStrangerInfo(session.userId, true)
    if (userInfo.level >= levelLimit) return false

    await botInternal(session).setGroupAddRequest(
      data.flag,
      data.sub_type,
      false,
      `等级不足${levelLimit}级`,
    )
    return true
  } catch (error) {
    eventLogger.error('获取用户信息失败:', error)
    return false
  }
}

async function acceptIfKeywordMatches(
  host: EventRuntimeHost,
  session: EventSession,
  data: RequestData,
  groupConfig: GroupConfig,
) {
  const approvalKeywords = Array.isArray(groupConfig.approvalKeywords)
    ? groupConfig.approvalKeywords
    : []
  const keywords = [...(host.config.keywords || []), ...approvalKeywords]

  if (!data.comment || !matchesAnyKeyword(data.comment, keywords)) return
  await botInternal(session).setGroupAddRequest(data.flag, data.sub_type, true)
}
