import type { WarnRecord } from '../../types'
import type { CacheData } from '../services/cache-types'
import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  type ConsoleGuildScope,
} from './console-guild-scope'
import { parseWarnKey } from './scope-filters'

export function registerWarnsAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/warns/reload', async function () {
    return handleWarnsReload(api, this)
  })
  api.addAuthorityListener('stuhelperGroupCenter/warns/list', async function (params?: { fetchNames?: boolean }) {
    const scope = await api.resolveConsoleScope(this)
    return success(buildWarnList(api, scope, Boolean(params?.fetchNames)))
  })
  api.addAuthorityListener('stuhelperGroupCenter/warns/update', async function (params: WarnUpdateParams) {
    return handleWarnUpdate(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/warns/add', async function (params: WarnAddParams) {
    return handleWarnAdd(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/warns/get', async function (params: { key: string }) {
    return handleWarnGet(api, this, params.key)
  })
  api.addAuthorityListener('stuhelperGroupCenter/warns/clear', async function (params: { key: string }) {
    return handleWarnClear(api, this, params.key)
  })
}

interface WarnUpdateParams {
  readonly key: string
  readonly count: number
}

interface WarnAddParams {
  readonly guildId: string
  readonly userId: string
}

interface WarnListItem {
  readonly key: string
  readonly guildId: string
  readonly userId: string
  readonly guildName: string
  readonly guildAvatar: string
  readonly userName: string
  readonly userAvatar: string
  readonly count: number
  readonly timestamp: number
}

async function handleWarnsReload(api: WebSocketAPIContext, client: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'warn reload')
    api.service.data.warns.reload()
    api.ctx.logger('stuhelperGroupCenter').info('警告数据已重新加载')
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '重新加载失败')
  }
}

function buildWarnList(api: WebSocketAPIContext, scope: ConsoleGuildScope, fetchNames: boolean): WarnListItem[] {
  const cacheData = fetchNames ? api.service.cache.getCachedData() : undefined
  const result: WarnListItem[] = []

  for (const [guildId, guildWarns] of Object.entries(api.service.data.warns.getAll())) {
    if (!isVisibleGuildWarns(scope, guildId, guildWarns)) continue
    for (const [userId, warnInfo] of Object.entries(guildWarns as WarnRecord)) {
      if (!isWarnInfo(warnInfo)) continue
      result.push(buildWarnListItem({ guildId, userId, info: warnInfo, cacheData }))
    }
  }
  return result
}

function isVisibleGuildWarns(scope: ConsoleGuildScope, guildId: string, guildWarns: unknown) {
  if (!guildWarns || typeof guildWarns !== 'object') return false
  return scope.kind === 'all' || scope.guildIds.has(guildId)
}

function isWarnInfo(value: unknown): value is { count: number, timestamp: number } {
  return Boolean(
    value
      && typeof value === 'object'
      && 'count' in value
      && typeof value.count === 'number'
      && 'timestamp' in value
      && typeof value.timestamp === 'number',
  )
}

function buildWarnListItem(input: {
  readonly guildId: string
  readonly userId: string
  readonly info: { count: number, timestamp: number }
  readonly cacheData?: CacheData
}): WarnListItem {
  const { guildId, userId, info, cacheData } = input
  const guildCache = cacheData?.guilds[guildId]
  const memberKey = `${guildId}:${userId}`
  const memberCache = cacheData?.members[memberKey]
  return {
    key: memberKey,
    guildId,
    userId,
    guildName: guildCache?.name || '',
    guildAvatar: guildCache?.avatar || (cacheData ? `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/` : ''),
    userName: memberCache?.nick || memberCache?.name || '',
    userAvatar: memberCache?.avatar || (cacheData ? `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640` : ''),
    count: info.count,
    timestamp: info.timestamp,
  }
}

async function handleWarnUpdate(api: WebSocketAPIContext, client: unknown, params: WarnUpdateParams) {
  try {
    const warnKey = await assertWarnRecordScope(api, client, params.key)
    const guildWarns = api.service.data.warns.get(warnKey.guildId)
    if (!guildWarns?.[warnKey.userId]) return error('Record not found')
    updateWarnRecord({
      api,
      guildId: warnKey.guildId,
      userId: warnKey.userId,
      count: params.count,
      guildWarns: guildWarns as WarnRecord,
    })
    await api.service.data.warns.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '更新警告失败')
  }
}

async function handleWarnAdd(api: WebSocketAPIContext, client: unknown, params: WarnAddParams) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, params.guildId, 'warn record')
    const current = (api.service.data.warns.get(params.guildId) || {}) as WarnRecord
    const previous = current[params.userId] || { count: 0, timestamp: 0 }
    api.service.data.warns.set(params.guildId, {
      ...current,
      [params.userId]: { count: previous.count + 1, timestamp: Date.now() },
    })
    await api.service.data.warns.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '添加警告失败')
  }
}

async function handleWarnGet(api: WebSocketAPIContext, client: unknown, key: string) {
  try {
    const warnKey = await assertWarnRecordScope(api, client, key)
    return success(api.service.data.warns.get(warnKey.guildId)?.[warnKey.userId] || null)
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取警告失败')
  }
}

async function handleWarnClear(api: WebSocketAPIContext, client: unknown, key: string) {
  try {
    const warnKey = await assertWarnRecordScope(api, client, key)
    removeWarnRecord(api, warnKey.guildId, warnKey.userId)
    await api.service.data.warns.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '清除警告失败')
  }
}

async function assertWarnRecordScope(api: WebSocketAPIContext, client: unknown, key: string) {
  const warnKey = parseWarnKey(key)
  if (!warnKey) throw new Error('Invalid key format')
  const scope = await api.resolveConsoleScope(client)
  assertConsoleGuildAccess(scope, warnKey.guildId, 'warn record')
  return warnKey
}

function updateWarnRecord(input: {
  readonly api: WebSocketAPIContext
  readonly guildId: string
  readonly userId: string
  readonly count: number
  readonly guildWarns: WarnRecord
}) {
  const { api, guildId, userId, count, guildWarns } = input
  if (count <= 0) {
    removeWarnRecord(api, guildId, userId)
    return
  }
  api.service.data.warns.set(guildId, {
    ...guildWarns,
    [userId]: { ...guildWarns[userId], count, timestamp: Date.now() },
  })
}

function removeWarnRecord(api: WebSocketAPIContext, guildId: string, userId: string) {
  const guildWarns = api.service.data.warns.get(guildId) as WarnRecord | undefined
  if (!guildWarns?.[userId]) return
  const nextWarns = { ...guildWarns }
  delete nextWarns[userId]
  if (Object.keys(nextWarns).length === 0) {
    api.service.data.warns.delete(guildId)
    return
  }
  api.service.data.warns.set(guildId, nextWarns)
}
