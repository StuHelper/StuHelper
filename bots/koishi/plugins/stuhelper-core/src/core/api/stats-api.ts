import { assertGlobalConsoleScope } from './console-guild-scope'
import { error, success } from './api-response'
import {
  filterGuildEntries,
  filterLogs,
  filterSubscriptions,
  type WebSocketAPIContext,
} from './websocket-api-context'
import {
  loadScopedBlacklistTotal,
  type MemberBlacklistStatsBackend,
} from './dashboard-blacklist-stats'

const DEFAULT_CHART_DAYS = 7
const MS_PER_DAY = 24 * 60 * 60 * 1000
const RANK_LIMIT = 10
const pkg = require('../../../package.json')

export function registerStatsAPI(
  api: WebSocketAPIContext,
  memberBlacklistBackend?: MemberBlacklistStatsBackend,
) {
  const { service, data, addAuthorityListener, resolveConsoleScope } = api

  addAuthorityListener('stuhelperGroupCenter/stats/modules', async function () {
    const scope = await resolveConsoleScope(this)
    assertGlobalConsoleScope(scope, 'module stats')
    return success(service.getAllModules().map(m => ({
      name: m.meta.name,
      description: m.meta.description,
      state: m.state,
      error: m.error ? m.error.message : undefined,
    })))
  })

  addAuthorityListener('stuhelperGroupCenter/stats/dashboard', async function () {
    const scope = await resolveConsoleScope(this)
    const scopedConfigs = filterGuildEntries(data.groupConfig.getAll(), scope)
    const scopedSubs = filterSubscriptions(data.subscriptions.get('list') || [], scope)
    return success({
      totalGroups: scopedConfigs.length,
      totalWarns: countWarns(data.warns.getAll(), scope),
      totalBlacklisted: await loadScopedBlacklistTotal(memberBlacklistBackend, scope),
      totalSubscriptions: scopedSubs.length,
      version: pkg.version,
      timestamp: Date.now(),
    })
  })

  addAuthorityListener('stuhelperGroupCenter/stats/charts', async function (params?: { days?: number }) {
    const scope = await resolveConsoleScope(this)
    const logModule = service.getAllModules().find(m => m.meta.name === 'log') as any
    if (!logModule) return error('Log module not found')
    const days = params?.days || DEFAULT_CHART_DAYS
    const stats = buildChartStats(filterLogs(await logModule.getAllLogs(), scope), days, service.cache.getCachedData())
    return success(stats)
  })
}

function countWarns(allWarns: Record<string, unknown>, scope: any) {
  let total = 0
  for (const [guildId, guildWarns] of Object.entries(allWarns)) {
    if (scope.kind !== 'all' && !scope.guildIds.has(guildId)) continue
    if (guildWarns && typeof guildWarns === 'object') {
      total += Object.keys(guildWarns).length
    }
  }
  return total
}

function buildChartStats(logs: any[], days: number, cacheData: any) {
  const now = Date.now()
  const dailyStats = initialDailyStats(now, days)
  const commandStats: Record<string, number> = {}
  const guildStats: Record<string, number> = {}
  const userStats: Record<string, { count: number, name: string }> = {}
  const successRate = { success: 0, fail: 0 }
  for (const log of logs) {
    collectLogStats(log, now - days * MS_PER_DAY, dailyStats, commandStats, guildStats, userStats, successRate)
  }
  return {
    trend: sortedCountRows(dailyStats, 'date'),
    distribution: topCountRows(commandStats, 'command'),
    successRate,
    guildRank: topGuildRows(guildStats, cacheData),
    userRank: topUserRows(userStats),
  }
}

function collectLogStats(log: any, startTime: number, dailyStats: Record<string, number>, commandStats: Record<string, number>, guildStats: Record<string, number>, userStats: Record<string, { count: number, name: string }>, successRate: { success: number; fail: number }) {
  const logTime = new Date(log.timestamp).getTime()
  if (Number.isNaN(logTime) || logTime < startTime) return
  const dateKey = new Date(log.timestamp).toISOString().slice(0, 10)
  if (dailyStats[dateKey] !== undefined) dailyStats[dateKey]++
  const cmd = log.command || 'unknown'
  commandStats[cmd] = (commandStats[cmd] || 0) + 1
  if (log.success) successRate.success++
  else successRate.fail++
  if (log.guildId) guildStats[log.guildId] = (guildStats[log.guildId] || 0) + 1
  if (log.userId) {
    userStats[log.userId] ||= { count: 0, name: log.username || log.userId }
    userStats[log.userId].count++
  }
}

function initialDailyStats(now: number, days: number) {
  const dailyStats: Record<string, number> = {}
  for (let i = 0; i < days; i++) {
    const date = new Date(now - i * MS_PER_DAY)
    dailyStats[date.toISOString().slice(0, 10)] = 0
  }
  return dailyStats
}

function sortedCountRows(stats: Record<string, number>, key: string) {
  return Object.entries(stats).sort(([a], [b]) => a.localeCompare(b)).map(([value, count]) => ({ [key]: value, count }))
}

function topCountRows(stats: Record<string, number>, key: string) {
  return Object.entries(stats).sort(([, a], [, b]) => b - a).slice(0, RANK_LIMIT)
    .map(([value, count]) => ({ [key]: value, count }))
}

function topGuildRows(stats: Record<string, number>, cacheData: any) {
  return Object.entries(stats).sort(([, a], [, b]) => b - a).slice(0, RANK_LIMIT)
    .map(([guildId, count]) => ({ guildId, count, name: cacheData.guilds[guildId]?.name || '' }))
}

function topUserRows(stats: Record<string, { count: number, name: string }>) {
  return Object.entries(stats).sort(([, a], [, b]) => b.count - a.count).slice(0, RANK_LIMIT)
    .map(([userId, data]) => ({ userId, count: data.count, name: data.name }))
}
