import type { WarnRecord } from '../../types'
import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'
import { error, success } from './api-response'
import {
  parseWarnKey,
  type ResolvedConsoleScope,
  type WebSocketAPIContext,
} from './websocket-api-context'

export function registerWarnsAPI(api: WebSocketAPIContext) {
  const { ctx, service, data, addAuthorityListener, resolveConsoleScope } = api

  addAuthorityListener('stuhelperGroupCenter/warns/reload', async function () {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'warn reload')
      data.warns.reload()
      ctx.logger('stuhelperGroupCenter').info('警告数据已重新加载')
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '重新加载失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/warns/list', async function (params?: { fetchNames?: boolean }) {
    const scope = await resolveConsoleScope(this)
    const cacheData = params?.fetchNames ? service.cache.getCachedData() : null
    return success(listWarnRecords(data.warns.getAll(), scope, cacheData))
  })

  addAuthorityListener('stuhelperGroupCenter/warns/update', async function (params: { key: string, count: number }) {
    try {
      const scope = await resolveConsoleScope(this)
      const warnKey = parseWarnKey(params.key)
      if (!warnKey) return error('Invalid key format')
      assertConsoleGuildAccess(scope, warnKey.guildId, 'warn record')
      return await updateWarnCount(api, warnKey.guildId, warnKey.userId, params.count)
    } catch (e) {
      return error(e instanceof Error ? e.message : '更新警告失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/warns/add', async function (params: { guildId: string, userId: string }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'warn record')
      await incrementWarnCount(api, params.guildId, params.userId)
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '添加警告失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/warns/get', async function (params: { key: string }) {
    try {
      const scope = await resolveConsoleScope(this)
      const warnKey = parseWarnKey(params.key)
      if (!warnKey) return error('Invalid key format')
      assertConsoleGuildAccess(scope, warnKey.guildId, 'warn record')
      return success(data.warns.get(warnKey.guildId)?.[warnKey.userId] || null)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取警告失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/warns/clear', async function (params: { key: string }) {
    try {
      const scope = await resolveConsoleScope(this)
      const warnKey = parseWarnKey(params.key)
      if (!warnKey) return error('Invalid key format')
      assertConsoleGuildAccess(scope, warnKey.guildId, 'warn record')
      await clearWarnRecord(api, warnKey.guildId, warnKey.userId)
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '清除警告失败')
    }
  })
}

function listWarnRecords(
  allWarns: Record<string, unknown>,
  scope: ResolvedConsoleScope,
  cacheData: any,
) {
  const result: any[] = []
  for (const [guildId, guildWarns] of Object.entries(allWarns)) {
    if (!guildWarns || typeof guildWarns !== 'object') continue
    if (scope.kind !== 'all' && !scope.guildIds.has(guildId)) continue
    pushGuildWarnRecords(result, guildId, guildWarns as WarnRecord, cacheData)
  }
  return result
}

function pushGuildWarnRecords(
  result: any[],
  guildId: string,
  guildWarns: WarnRecord,
  cacheData: any,
) {
  for (const [userId, warnInfo] of Object.entries(guildWarns)) {
    if (!warnInfo || typeof warnInfo !== 'object' || !('count' in warnInfo)) continue
    const memberKey = `${guildId}:${userId}`
    const memberCache = cacheData?.members[memberKey]
    result.push({
      key: memberKey,
      guildId,
      userId,
      guildName: cacheData?.guilds[guildId]?.name || '',
      guildAvatar: cacheData?.guilds[guildId]?.avatar || (cacheData ? guildAvatar(guildId) : ''),
      userName: memberCache?.nick || memberCache?.name || '',
      userAvatar: memberCache?.avatar || (cacheData ? qqAvatar(userId) : ''),
      count: warnInfo.count,
      timestamp: warnInfo.timestamp,
    })
  }
}

async function updateWarnCount(api: WebSocketAPIContext, guildId: string, userId: string, count: number) {
  const guildWarns = api.data.warns.get(guildId)
  if (!guildWarns?.[userId]) return error('Record not found')
  if (count <= 0) {
    await removeWarnRecord(api, guildId, userId, guildWarns as WarnRecord)
    return success({ success: true })
  }
  guildWarns[userId].count = count
  guildWarns[userId].timestamp = Date.now()
  api.data.warns.set(guildId, guildWarns as WarnRecord)
  await api.data.warns.flush()
  return success({ success: true })
}

async function incrementWarnCount(api: WebSocketAPIContext, guildId: string, userId: string) {
  const guildWarns = api.data.warns.get(guildId) || {}
  guildWarns[userId] ||= { count: 0, timestamp: 0 }
  guildWarns[userId].count++
  guildWarns[userId].timestamp = Date.now()
  api.data.warns.set(guildId, guildWarns as WarnRecord)
  await api.data.warns.flush()
}

async function clearWarnRecord(api: WebSocketAPIContext, guildId: string, userId: string) {
  const guildWarns = api.data.warns.get(guildId)
  if (guildWarns?.[userId]) {
    await removeWarnRecord(api, guildId, userId, guildWarns as WarnRecord)
  }
}

async function removeWarnRecord(api: WebSocketAPIContext, guildId: string, userId: string, guildWarns: WarnRecord) {
  delete guildWarns[userId]
  if (Object.keys(guildWarns).length === 0) {
    api.data.warns.delete(guildId)
  } else {
    api.data.warns.set(guildId, guildWarns)
  }
  await api.data.warns.flush()
}

function qqAvatar(userId: string) {
  return `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
}

function guildAvatar(guildId: string) {
  return `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/`
}
