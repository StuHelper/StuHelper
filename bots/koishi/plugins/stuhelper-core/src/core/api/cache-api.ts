import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'

export function registerCacheAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/cache/stats', async function () {
    return handleCacheStats(api, this)
  })
  api.addAuthorityListener('stuhelperGroupCenter/cache/refresh', async function () {
    return handleCacheRefresh(api, this)
  })
  api.addAuthorityListener('stuhelperGroupCenter/cache/clear', async function () {
    return handleCacheClear(api, this)
  })
  api.addAuthorityListener('stuhelperGroupCenter/cache/fetch-name', async function (params: CacheFetchNameParams) {
    return handleCacheFetchName(api, this, params)
  })
}

interface CacheFetchNameParams {
  readonly type: 'guild' | 'user' | 'member'
  readonly guildId?: string
  readonly userId?: string
}

async function handleCacheStats(api: WebSocketAPIContext, client: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'cache stats')
    return success(api.service.cache.getStats())
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('获取缓存统计失败:', cause)
    return error(cause instanceof Error ? cause.message : '获取缓存统计失败')
  }
}

async function handleCacheRefresh(api: WebSocketAPIContext, client: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'cache refresh')
    api.ctx.logger('stuhelperGroupCenter').info('开始刷新缓存...')
    await api.service.cache.refreshAll()
    api.ctx.logger('stuhelperGroupCenter').info('缓存刷新完成')
    return success({ success: true, stats: api.service.cache.getStats() })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('刷新缓存失败:', cause)
    return error(cause instanceof Error ? cause.message : '刷新缓存失败')
  }
}

async function handleCacheClear(api: WebSocketAPIContext, client: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'cache clear')
    await api.service.cache.clearAll()
    api.ctx.logger('stuhelperGroupCenter').info('缓存已清空')
    return success({ success: true })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('清空缓存失败:', cause)
    return error(cause instanceof Error ? cause.message : '清空缓存失败')
  }
}

async function handleCacheFetchName(api: WebSocketAPIContext, client: unknown, params: CacheFetchNameParams) {
  try {
    const scope = await api.resolveConsoleScope(client)
    if (params.type === 'guild' && params.guildId) {
      assertConsoleGuildAccess(scope, params.guildId, 'cache guild name')
      const info = await api.service.cache.getGuildInfo(params.guildId)
      return success({ name: info?.name || '', avatar: info?.avatar })
    }
    if (params.type === 'user' && params.userId) {
      assertGlobalConsoleScope(scope, 'cache user name')
      const info = await api.service.cache.getUserInfo(params.userId)
      return success({ name: info?.name || '', avatar: info?.avatar })
    }
    if (params.type === 'member' && params.guildId && params.userId) {
      assertConsoleGuildAccess(scope, params.guildId, 'cache member name')
      const info = await api.service.cache.getMemberInfo(params.guildId, params.userId)
      return success({ name: info?.name || '', nick: info?.nick || '', avatar: info?.avatar })
    }
    return error('无效的参数')
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取名称失败')
  }
}
