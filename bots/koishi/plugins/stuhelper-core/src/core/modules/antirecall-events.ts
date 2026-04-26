import type { Session } from 'koishi'

import type { RecalledMessage } from '../../types'
import type { AntiRecallModule, CachedMessage } from './antirecall.module'

const MESSAGE_CACHE_EXPIRATION_MS = 5 * 60 * 1000
const CLEANUP_INTERVAL_MS = 24 * 60 * 60 * 1000
const DEFAULT_RETENTION_DAYS = 7
const DEFAULT_MAX_RECORDS_PER_USER = 100
const RANDOM_ID_RADIX = 36
const RANDOM_ID_START = 2
const RANDOM_ID_END = 11
const MISSING_CONTENT_MESSAGE = '[无法获取消息内容 - 消息发送时间过久或在机器人离线/重启期间发送]'

interface RecallContent {
  content: string
  username: string
  originalTimestamp: number
}

export function registerAntiRecallEventListeners(host: AntiRecallModule): void {
  host.ctx.on('message', (session) => {
    cacheMessage(host, session)
  })

  host.ctx.on('message-deleted', async (session) => {
    await handleMessageRecall(host, session)
  })
}

export function scheduleAntiRecallCleanup(host: AntiRecallModule): void {
  cleanExpiredRecallRecords(host)
  host.setCleanupInterval(setInterval(() => {
    cleanExpiredRecallRecords(host)
  }, CLEANUP_INTERVAL_MS))
}

function cacheMessage(host: AntiRecallModule, session: Session): void {
  if (!session.guildId) return
  if (!host.isEnabledForGuild(session.guildId)) return
  if (!session.messageId || !session.content) return

  host.cacheMessage(session.messageId, {
    content: session.content,
    userId: session.userId,
    username: getAuthorName(session),
    timestamp: Date.now(),
  })
  setTimeout(() => host.deleteCachedMessage(session.messageId), MESSAGE_CACHE_EXPIRATION_MS)
}

async function handleMessageRecall(host: AntiRecallModule, session: Session): Promise<void> {
  if (session.userId === session.selfId) return
  if (!session.guildId || !host.isEnabledForGuild(session.guildId)) return

  const recallContent = resolveRecallContent(host, session)
  const recalledMessage = createRecalledMessage(session, recallContent)
  saveRecalledMessage(host, recalledMessage)
  await sendRecallNotification(host, session, recalledMessage)
}

function resolveRecallContent(host: AntiRecallModule, session: Session): RecallContent {
  const cachedMessage = host.getCachedMessage(session.messageId)
  if (!cachedMessage) {
    return {
      content: MISSING_CONTENT_MESSAGE,
      username: getAuthorName(session) || 'Unknown',
      originalTimestamp: Date.now(),
    }
  }

  host.deleteCachedMessage(session.messageId)
  return {
    content: cachedMessage.content,
    username: cachedMessage.username,
    originalTimestamp: cachedMessage.timestamp,
  }
}

function createRecalledMessage(
  session: Session,
  recallContent: RecallContent,
): RecalledMessage {
  return {
    id: `${Date.now()}_${Math.random().toString(RANDOM_ID_RADIX).substring(RANDOM_ID_START, RANDOM_ID_END)}`,
    messageId: session.messageId,
    userId: session.userId || 'unknown',
    username: recallContent.username,
    guildId: session.guildId,
    channelId: session.channelId,
    content: recallContent.content,
    timestamp: recallContent.originalTimestamp,
    recallTime: Date.now(),
    elements: session.elements || [],
  }
}

function saveRecalledMessage(host: AntiRecallModule, recalledMessage: RecalledMessage): void {
  const records = host.data.recallRecords.getAll()
  const { guildId, userId } = recalledMessage
  const config = host.getAntiRecallConfig(guildId)

  if (!records[guildId]) records[guildId] = {}
  if (!records[guildId][userId]) records[guildId][userId] = []

  records[guildId][userId].unshift(recalledMessage)
  const maxRecords = config?.maxRecordsPerUser || DEFAULT_MAX_RECORDS_PER_USER
  if (records[guildId][userId].length > maxRecords) {
    records[guildId][userId] = records[guildId][userId].slice(0, maxRecords)
  }

  host.data.recallRecords.setAll(records)
}

async function sendRecallNotification(
  host: AntiRecallModule,
  session: Session,
  recalledMessage: RecalledMessage,
): Promise<void> {
  try {
    const config = host.getAntiRecallConfig(session.guildId)
    const timeStr = config?.showOriginalTime
      ? new Date(recalledMessage.timestamp).toLocaleString('zh-CN')
      : ''
    let notification = `检测到撤回消息\n`

    notification += `用户: ${recalledMessage.username}(${recalledMessage.userId})\n`
    if (timeStr) {
      notification += `发送时间: ${timeStr}\n`
    }
    notification += `内容: ${recalledMessage.content}`
    await host.ctx.stuhelperGroupCenter.pushMessage(session.bot, notification, 'antiRecall')
  } catch (error) {
    host.data.writeLog(`[antirecall] 发送撤回通知失败: ${error}`)
  }
}

function cleanExpiredRecallRecords(host: AntiRecallModule): void {
  try {
    const records = host.data.recallRecords.getAll()
    let hasChanges = false

    for (const guildId in records) {
      const retentionDays = host.getAntiRecallConfig(guildId)?.retentionDays || DEFAULT_RETENTION_DAYS
      const cutoffTime = Date.now() - (retentionDays * CLEANUP_INTERVAL_MS)
      hasChanges = pruneGuildRecords(records[guildId], cutoffTime) || hasChanges
      if (Object.keys(records[guildId]).length === 0) delete records[guildId]
    }

    if (hasChanges) {
      host.data.recallRecords.setAll(records)
      host.logInfo('已清理过期的撤回记录')
    }
  } catch (error) {
    host.data.writeLog(`[antirecall] 清理过期撤回记录失败: ${error}`)
  }
}

function pruneGuildRecords(
  guildRecords: Record<string, RecalledMessage[]>,
  cutoffTime: number,
): boolean {
  let hasChanges = false
  for (const userId in guildRecords) {
    const originalLength = guildRecords[userId].length
    guildRecords[userId] = guildRecords[userId].filter(record => record.recallTime > cutoffTime)
    if (guildRecords[userId].length !== originalLength) hasChanges = true
    if (guildRecords[userId].length === 0) delete guildRecords[userId]
  }
  return hasChanges
}

function getAuthorName(session: Session): string {
  return session.author?.name || session.author?.nick || `用户${session.userId}`
}
