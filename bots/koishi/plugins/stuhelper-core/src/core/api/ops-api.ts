import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'
import { error, success } from './api-response'
import {
  filterLogs,
  type WebSocketAPIContext,
} from './websocket-api-context'

const DEFAULT_LOG_PAGE = 1
const DEFAULT_LOG_PAGE_SIZE = 20

interface LogSearchParams {
  startTime?: string | number
  endTime?: string | number
  command?: string
  userId?: string
  username?: string
  details?: string
  guildId?: string
  page?: number
  pageSize?: number
}

export function registerOpsAPI(api: WebSocketAPIContext) {
  registerLogsAPI(api)
  registerSettingsAPI(api)
  registerCacheAPI(api)
}

function registerLogsAPI(api: WebSocketAPIContext) {
  const { service, addAuthorityListener, resolveConsoleScope } = api
  addAuthorityListener('stuhelperGroupCenter/logs/search', async function (params: LogSearchParams) {
    try {
      const scope = await resolveConsoleScope(this)
      if (params.guildId) assertConsoleGuildAccess(scope, params.guildId, 'log search')
      const logModule = service.getAllModules().find(m => m.meta.name === 'log') as any
      if (!logModule) return error('Log module not found')
      const logs = filterLogs(await logModule.getAllLogs(), scope).filter((log) => matchesLog(log, params))
      const page = params.page || DEFAULT_LOG_PAGE
      const pageSize = params.pageSize || DEFAULT_LOG_PAGE_SIZE
      return success({ list: paginate(logs, page, pageSize), total: logs.length, page, pageSize })
    } catch (e) {
      return error(e instanceof Error ? e.message : '检索日志失败')
    }
  })
}

function registerSettingsAPI(api: WebSocketAPIContext) {
  const { ctx, service, addAuthorityListener, resolveConsoleScope } = api
  addAuthorityListener('stuhelperGroupCenter/settings/get', async function () {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'settings')
      return success(service.settings.settings)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取设置失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/settings/update', async function (params: { settings: any }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'settings')
      if (!params.settings || typeof params.settings !== 'object') return error('无效的设置数据')
      await service.settings.update(params.settings)
      ctx.logger('stuhelperGroupCenter').info('设置已更新')
      return success({ success: true })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('更新设置失败:', e)
      return error(e instanceof Error ? e.message : '更新设置失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/settings/reset', async function () {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'settings')
      await service.settings.reset()
      ctx.logger('stuhelperGroupCenter').info('设置已重置为默认值')
      return success({ success: true })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('重置设置失败:', e)
      return error(e instanceof Error ? e.message : '重置设置失败')
    }
  })
}

function registerCacheAPI(api: WebSocketAPIContext) {
  const { ctx, service, addAuthorityListener, resolveConsoleScope } = api
  addAuthorityListener('stuhelperGroupCenter/cache/stats', async function () {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'cache stats')
      return success(service.cache.getStats())
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('获取缓存统计失败:', e)
      return error(e instanceof Error ? e.message : '获取缓存统计失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/cache/refresh', async function () {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'cache refresh')
      ctx.logger('stuhelperGroupCenter').info('开始刷新缓存...')
      await service.cache.refreshAll()
      ctx.logger('stuhelperGroupCenter').info('缓存刷新完成')
      return success({ success: true, stats: service.cache.getStats() })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('刷新缓存失败:', e)
      return error(e instanceof Error ? e.message : '刷新缓存失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/cache/clear', async function () {
    try {
      const scope = await resolveConsoleScope(this)
      assertGlobalConsoleScope(scope, 'cache clear')
      await service.cache.clearAll()
      ctx.logger('stuhelperGroupCenter').info('缓存已清空')
      return success({ success: true })
    } catch (e) {
      ctx.logger('stuhelperGroupCenter').error('清空缓存失败:', e)
      return error(e instanceof Error ? e.message : '清空缓存失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/cache/fetch-name', async function (params: { type: 'guild' | 'user' | 'member'; guildId?: string; userId?: string }) {
    return fetchCachedName(api, params, this)
  })
}

async function fetchCachedName(
  api: WebSocketAPIContext,
  params: { type: 'guild' | 'user' | 'member'; guildId?: string; userId?: string },
  client: unknown,
) {
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
  } catch (e) {
    return error(e instanceof Error ? e.message : '获取名称失败')
  }
}

function matchesLog(log: any, params: LogSearchParams) {
  try {
    const time = new Date(log.timestamp).getTime()
    if (params.startTime && time < new Date(params.startTime).getTime()) return false
    if (params.endTime && time > new Date(params.endTime).getTime()) return false
    if (params.command && !String(log.command || '').toLowerCase().includes(params.command.toLowerCase())) return false
    if (params.userId && String(log.userId) !== params.userId) return false
    if (params.username && !String(log.username || '').toLowerCase().includes(params.username.toLowerCase())) return false
    if (params.guildId && String(log.guildId) !== params.guildId) return false
    return matchesLogDetails(log, params.details)
  } catch {
    return false
  }
}

function matchesLogDetails(log: any, details: string | undefined) {
  if (!details) return true
  const keyword = details.toLowerCase()
  return String(log.result || '').toLowerCase().includes(keyword)
    || String(log.error || '').toLowerCase().includes(keyword)
    || log.args?.some((arg: any) => String(arg).toLowerCase().includes(keyword))
    || JSON.stringify(log.options || {}).toLowerCase().includes(keyword)
}

function paginate<T>(items: readonly T[], page: number, pageSize: number) {
  return items.slice((page - 1) * pageSize, page * pageSize)
}
