import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope, type ConsoleGuildScope } from './console-guild-scope'
import { loadScopedBlacklistTotal } from './dashboard-blacklist-stats'
import { findLogModule } from './log-module-lookup'
import { filterGuildEntries, filterLogs, filterSubscriptions } from './scope-filters'
import type { CommandLogRecord } from '../modules/log.module'

const DEFAULT_CHART_DAYS = 7
const MS_PER_DAY = 24 * 60 * 60 * 1000
const ISO_DATE_LENGTH = 10
const RANK_LIMIT = 10

export function registerStatsAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/stats/modules', async function () {
    const scope = await api.resolveConsoleScope(this)
    assertGlobalConsoleScope(scope, 'module stats')
    return success(api.service.getAllModules().map((module) => ({
      name: module.meta.name,
      description: module.meta.description,
      state: module.state,
      error: module.error ? module.error.message : undefined,
    })))
  })
  api.addAuthorityListener('stuhelperGroupCenter/stats/dashboard', async function () {
    const scope = await api.resolveConsoleScope(this)
    return success(await buildDashboardStats(api, scope))
  })
  api.addAuthorityListener('stuhelperGroupCenter/stats/charts', async function (params?: { days?: number }) {
    const scope = await api.resolveConsoleScope(this)
    return handleChartStats(api, scope, params)
  })
}

async function buildDashboardStats(api: WebSocketAPIContext, scope: ConsoleGuildScope) {
  const allWarns = api.service.data.warns.getAll()
  const scopedConfigs = filterGuildEntries(api.service.data.groupConfig.getAll(), scope)
  const subscriptions = api.service.data.subscriptions.get('list') || []
  const scopedSubs = filterSubscriptions(subscriptions, scope)

  return {
    totalGroups: scopedConfigs.length,
    totalWarns: countScopedWarns(allWarns, scope),
    totalBlacklisted: await loadScopedBlacklistTotal(api.platform, scope),
    totalSubscriptions: scopedSubs.length,
    version: api.packageVersion,
    timestamp: Date.now(),
  }
}

function countScopedWarns(allWarns: Record<string, unknown>, scope: ConsoleGuildScope) {
  let totalWarnCount = 0
  for (const [guildId, guildWarns] of Object.entries(allWarns)) {
    if (scope.kind !== 'all' && !scope.guildIds.has(guildId)) continue
    if (guildWarns && typeof guildWarns === 'object') {
      totalWarnCount += Object.keys(guildWarns).length
    }
  }
  return totalWarnCount
}

async function handleChartStats(api: WebSocketAPIContext, scope: ConsoleGuildScope, params?: { days?: number }) {
  const days = params?.days || DEFAULT_CHART_DAYS
  const logModule = findLogModule(api)
  if (!logModule) return error('Log module not found')

  const logs = filterLogs(await logModule.getAllLogs(), scope)
  const now = Date.now()
  const stats = collectChartStats(logs, { now, days })
  return success({
    trend: buildTrend(stats.dailyStats),
    distribution: buildCommandRank(stats.commandStats),
    successRate: { success: stats.successCount, fail: stats.failCount },
    guildRank: buildGuildRank(api, stats.guildStats),
    userRank: buildUserRank(stats.userStats),
  })
}

interface ChartStatsState {
  dailyStats: Record<string, number>
  commandStats: Record<string, number>
  guildStats: Record<string, number>
  userStats: Record<string, { count: number, name: string }>
  counters: {
    successCount: number
    failCount: number
  }
  startTime: number
}

function collectChartStats(logs: CommandLogRecord[], options: { now: number, days: number }) {
  const dailyStats = initDailyStats(options.now, options.days)
  const commandStats: Record<string, number> = {}
  const guildStats: Record<string, number> = {}
  const userStats: Record<string, { count: number, name: string }> = {}
  const counters = { successCount: 0, failCount: 0 }
  const startTime = options.now - options.days * MS_PER_DAY

  logs.forEach((log) => applyLogToChartStats(log, { dailyStats, commandStats, guildStats, userStats, counters, startTime }))
  return { dailyStats, commandStats, guildStats, userStats, ...counters }
}

function initDailyStats(now: number, days: number) {
  const dailyStats: Record<string, number> = {}
  for (let i = 0; i < days; i++) {
    const date = new Date(now - i * MS_PER_DAY)
    dailyStats[date.toISOString().slice(0, ISO_DATE_LENGTH)] = 0
  }
  return dailyStats
}

function applyLogToChartStats(log: CommandLogRecord, stats: ChartStatsState) {
  const logTime = new Date(log.timestamp).getTime()
  if (logTime < stats.startTime) return

  const dateKey = new Date(log.timestamp).toISOString().slice(0, ISO_DATE_LENGTH)
  if (stats.dailyStats[dateKey] !== undefined) stats.dailyStats[dateKey]++
  const command = log.command || 'unknown'
  stats.commandStats[command] = (stats.commandStats[command] || 0) + 1
  if (log.success) stats.counters.successCount++
  else stats.counters.failCount++
  if (log.guildId) stats.guildStats[log.guildId] = (stats.guildStats[log.guildId] || 0) + 1
  if (log.userId) incrementUserStats(stats.userStats, log)
}

function incrementUserStats(userStats: Record<string, { count: number, name: string }>, log: CommandLogRecord) {
  if (!userStats[log.userId]) {
    userStats[log.userId] = { count: 0, name: log.username || log.userId }
  }
  userStats[log.userId].count++
}

function buildTrend(dailyStats: Record<string, number>) {
  return Object.entries(dailyStats)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([date, count]) => ({ date, count }))
}

function buildCommandRank(stats: Record<string, number>) {
  return Object.entries(stats)
    .sort(([, a], [, b]) => b - a)
    .slice(0, RANK_LIMIT)
    .map(([command, count]) => ({ command, count }))
}

function buildGuildRank(api: WebSocketAPIContext, guildStats: Record<string, number>) {
  const cacheData = api.service.cache.getCachedData()
  return Object.entries(guildStats)
    .sort(([, a], [, b]) => b - a)
    .slice(0, RANK_LIMIT)
    .map(([guildId, count]) => ({ guildId, count, name: cacheData.guilds[guildId]?.name || '' }))
}

function buildUserRank(userStats: Record<string, { count: number, name: string }>) {
  return Object.entries(userStats)
    .sort(([, a], [, b]) => b.count - a.count)
    .slice(0, RANK_LIMIT)
    .map(([userId, data]) => ({ userId, count: data.count, name: data.name }))
}
