import type { GroupConfig } from '../../types'
import { PlatformAPIError } from '@stuhelper/koishi-shared'
import {
  DEFAULT_LEVEL_LIMIT,
  MEMBER_BLACKLIST_ACCESS_TIMEOUT_MS,
  DEFAULT_MEMBER_REQUEST_CONFIG,
  admissionBusinessPlatformOf,
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
  if (await rejectDuringLeaveCooldown(host, session)) return

  const groupConfig = groupConfigOf(host, session.guildId, DEFAULT_MEMBER_REQUEST_CONFIG)
  if (await rejectBelowLevelLimit(session, groupConfig)) return

  if (await rejectBlacklistedGroupRequest(host, {
    session,
    failureLog: '拒绝入群申请失败:',
  })) return
  if (await tryAcceptByAdmissionPolicy(host, session)) return
  if (await acceptIfKeywordMatches(host, session, groupConfig)) return
}

async function rejectBlacklistedFriendRequest(
  host: EventRuntimeHost,
  session: EventSession,
): Promise<boolean> {
  if (!await isBackendBlacklisted(host, session)) return false

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
  if (!await isBackendBlacklisted(host, request.session)) return false

  try {
    await rejectGuildOrMemberRequest(request.session, '您在黑名单中')
  } catch (error) {
    eventLogger.error(request.failureLog, error)
  }
  return true
}

async function isBackendBlacklisted(host: EventRuntimeHost, session: EventSession): Promise<boolean> {
  if (!host.admissionPlatform) {
    throw new Error('admission platform client is required for blacklist access')
  }
  try {
    const access = await host.admissionPlatform.getMemberBlacklistAccess({
      platform: admissionBusinessPlatformOf(session.platform),
      guildID: session.guildId,
      subjectType: 'qq_user',
      subjectID: session.userId,
    }, { timeoutMs: MEMBER_BLACKLIST_ACCESS_TIMEOUT_MS })
    return !access.canJoin || access.decision === 'blocked'
  } catch (error) {
    eventLogger.warn('成员黑名单准入检查失败，跳过请求阶段黑名单裁决:', error)
    return false
  }
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
): Promise<boolean> {
  const approvalKeywords = Array.isArray(groupConfig.approvalKeywords)
    ? groupConfig.approvalKeywords
    : []
  const keywords = [...(host.config.keywords || []), ...approvalKeywords]
  const comment = requestCommentOf(session)

  if (!comment || !matchesAnyKeyword(comment, keywords)) return false
  await session.bot.handleGuildMemberRequest(requestIdOf(session), true)
  return true
}

async function tryAcceptByAdmissionPolicy(host: EventRuntimeHost, session: EventSession): Promise<boolean> {
  if (!host.admissionPlatform) {
    return false
  }
  try {
    await acceptByAdmissionPolicy(host, session)
    return true
  } catch (error) {
    if (isAdmissionPolicyNotFound(error)) {
      return false
    }
    throw error
  }
}

async function acceptByAdmissionPolicy(host: EventRuntimeHost, session: EventSession): Promise<void> {
  if (!host.admissionPlatform) {
    throw new Error('admission platform client is required for guild-member-request auto-approve')
  }
  const requestID = requestIdOf(session)
  const platform = admissionBusinessPlatformOf(session.platform)
  const decision = await host.admissionPlatform.resolveJoinRequestDecision({
    platform,
    guildID: session.guildId,
    qqID: session.userId,
    requestID,
    rawEvent: rawRequestEvent(session),
  })
  const approve = decision.decision === 'approve'
  try {
    await session.bot.handleGuildMemberRequest(requestID, approve, approve ? undefined : decision.reason)
    await recordAdmissionJoinRequest({
      host,
      session,
      platform,
      decision: decision.decision,
      success: true,
    })
  } catch (error) {
    await recordAdmissionJoinRequest({
      host,
      session,
      platform,
      decision: decision.decision,
      success: false,
      error: errorMessage(error),
    })
    throw error
  }
}

function isAdmissionPolicyNotFound(error: unknown): boolean {
  return error instanceof PlatformAPIError && error.status === 404
}

async function recordAdmissionJoinRequest(input: {
  readonly host: EventRuntimeHost
  readonly session: EventSession
  readonly platform: string
  readonly decision: 'approve' | 'reject'
  readonly success: boolean
  readonly error?: string
}) {
  const { host, session, platform, decision, success, error } = input
  if (!host.admissionPlatform) {
    throw new Error('admission platform client is required to record join request events')
  }
  const event = {
    platform,
    guildID: session.guildId,
    qqID: session.userId,
    requestID: requestIdOf(session),
    decision,
    success,
    rawEvent: rawRequestEvent(session),
  }
  if (error) {
    await host.admissionPlatform.recordJoinRequestEvent({ ...event, error })
    return
  }
  await host.admissionPlatform.recordJoinRequestEvent({
    ...event,
  })
}

function rawRequestEvent(session: EventSession) {
  return (session.event as { _data?: Record<string, unknown> } | undefined)?._data || {}
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

async function rejectGuildOrMemberRequest(session: EventSession, message: string): Promise<void> {
  if (session.type === 'guild-request') {
    await session.bot.handleGuildRequest(requestIdOf(session), false, message)
    return
  }
  await session.bot.handleGuildMemberRequest(requestIdOf(session), false, message)
}
