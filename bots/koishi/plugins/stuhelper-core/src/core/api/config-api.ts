import { error, success } from './api-response'
import type { WebSocketAPIContext } from './api-context'
import { assertConsoleGuildAccess } from './console-guild-scope'
import { filterGuildEntries } from './scope-filters'
import type { GroupConfig } from '../../types'

const DEFAULT_FORBIDDEN_MUTE_DURATION = 600000
const DEFAULT_DICE_LENGTH_LIMIT = 1000
const DEFAULT_BANME_BASE_MIN = 1
const DEFAULT_BANME_BASE_MAX = 30
const DEFAULT_BANME_GROWTH_RATE = 30
const DEFAULT_BANME_JACKPOT_PROBABILITY = 0.006
const DEFAULT_BANME_SOFT_PITY = 73
const DEFAULT_BANME_HARD_PITY = 89
const DEFAULT_BANME_UP_DURATION = '24h'
const DEFAULT_BANME_LOSE_DURATION = '12h'
const MAX_CONFIG_DEPTH = 8
const FORBIDDEN_CONFIG_KEYS = new Set(['__proto__', 'prototype', 'constructor'])

interface ConfigUpdateParams {
  readonly guildId: string
  readonly config: GroupConfig
}

export function registerConfigAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/config/reload', async () => handleConfigReload(api))
  api.addAuthorityListener('stuhelperGroupCenter/config/list', async function (params?: { fetchNames?: boolean }) {
    const scope = await api.resolveConsoleScope(this)
    return success(buildConfigList(api, filterGuildEntries(api.service.data.groupConfig.getAll(), scope), params))
  })
  api.addAuthorityListener('stuhelperGroupCenter/config/get', async function (params: { guildId: string }) {
    return handleConfigGet(api, this, params.guildId)
  })
  api.addAuthorityListener('stuhelperGroupCenter/config/update', async function (params: unknown) {
    return handleConfigUpdate(api, this, params)
  })
  api.addAuthorityListener('stuhelperGroupCenter/config/create', async function (params: { guildId: string }) {
    return handleConfigCreate(api, this, params.guildId)
  })
  api.addAuthorityListener('stuhelperGroupCenter/config/delete', async function (params: { guildId: string }) {
    const scope = await api.resolveConsoleScope(this)
    assertConsoleGuildAccess(scope, params.guildId, 'group config')
    api.service.data.groupConfig.delete(params.guildId)
    await api.service.data.groupConfig.flush()
    return success({ success: true })
  })
}

async function handleConfigReload(api: WebSocketAPIContext) {
  try {
    const groupConfig = api.service.data.groupConfig
    groupConfig.reload()
    const count = Object.keys(groupConfig.getAll()).length
    api.ctx.logger('stuhelperGroupCenter').info('群组配置已重新加载，共 %d 条', count)
    return success({ success: true, count })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('重新加载配置失败:', cause)
    return error(cause instanceof Error ? cause.message : '重新加载失败')
  }
}

function buildConfigList(
  api: WebSocketAPIContext,
  scopedConfigs: [string, GroupConfig][],
  params?: { fetchNames?: boolean },
) {
  const results: Record<string, GroupConfig & { guildName: string; guildAvatar: string }> = {}
  const cacheData = params?.fetchNames ? api.service.cache.getCachedData() : undefined

  scopedConfigs.forEach(([guildId, config]) => {
    const cached = cacheData?.guilds[guildId]
    results[guildId] = {
      ...config,
      guildName: cached?.name || '',
      guildAvatar: cached?.avatar || (cacheData ? `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/` : ''),
    }
  })
  return results
}

async function handleConfigGet(api: WebSocketAPIContext, client: unknown, guildId: string) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, guildId, 'group config')
    return success(api.service.data.groupConfig.get(guildId))
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取群组配置失败')
  }
}

async function handleConfigUpdate(api: WebSocketAPIContext, client: unknown, input: unknown) {
  try {
    const params = parseConfigUpdateParams(input)
    const scope = await api.resolveConsoleScope(client)
    assertConsoleGuildAccess(scope, params.guildId, 'group config')
    api.service.data.groupConfig.set(params.guildId, params.config)
    await api.service.data.groupConfig.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '更新群组配置失败')
  }
}

async function handleConfigCreate(api: WebSocketAPIContext, client: unknown, guildId: string) {
  const scope = await api.resolveConsoleScope(client)
  assertConsoleGuildAccess(scope, guildId, 'group config')
  if (api.service.data.groupConfig.get(guildId)) {
    return error('配置已存在')
  }
  api.service.data.groupConfig.set(guildId, createDefaultGroupConfig())
  await api.service.data.groupConfig.flush()
  return success({ success: true })
}

function parseConfigUpdateParams(input: unknown): ConfigUpdateParams {
  if (!isPlainRecord(input)) {
    throw new Error('group config update input must be an object')
  }

  const guildId = readNonEmptyString(input.guildId, 'guildId')
  return {
    guildId,
    config: normalizeGroupConfig(input.config),
  }
}

function normalizeGroupConfig(input: unknown): GroupConfig {
  if (!isPlainRecord(input)) {
    throw new Error('group config must be an object')
  }
  return normalizeConfigValue(input, 'config', 0) as GroupConfig
}

function normalizeConfigValue(input: unknown, path: string, depth: number): unknown {
  if (depth > MAX_CONFIG_DEPTH) {
    throw new Error(`${path} is too deeply nested`)
  }
  if (input === null || typeof input === 'string' || typeof input === 'boolean') {
    return input
  }
  if (typeof input === 'number') {
    if (!Number.isFinite(input)) throw new Error(`${path} must be a finite number`)
    return input
  }
  if (Array.isArray(input)) {
    return input.map((item, index) => normalizeConfigValue(item, `${path}[${index}]`, depth + 1))
  }
  if (isPlainRecord(input)) {
    const result: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(input)) {
      if (FORBIDDEN_CONFIG_KEYS.has(key)) {
        throw new Error(`${path} contains unsupported key: ${key}`)
      }
      if (value === undefined) continue
      result[key] = normalizeConfigValue(value, `${path}.${key}`, depth + 1)
    }
    return result
  }
  throw new Error(`${path} contains unsupported value type`)
}

function readNonEmptyString(value: unknown, field: string): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${field} must be a non-empty string`)
  }
  return value.trim()
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return false
  const prototype = Object.getPrototypeOf(value)
  return prototype === Object.prototype || prototype === null
}

function createDefaultGroupConfig() {
  return {
    welcomeEnabled: false,
    antiRecall: { enabled: false },
    antiRepeat: { enabled: false, threshold: 3 },
    forbidden: { autoDelete: false, autoBan: false, autoKick: false, muteDuration: DEFAULT_FORBIDDEN_MUTE_DURATION },
    dice: { enabled: true, lengthLimit: DEFAULT_DICE_LENGTH_LIMIT },
    banme: {
      enabled: true,
      baseMin: DEFAULT_BANME_BASE_MIN,
      baseMax: DEFAULT_BANME_BASE_MAX,
      growthRate: DEFAULT_BANME_GROWTH_RATE,
      jackpot: {
        enabled: true,
        baseProb: DEFAULT_BANME_JACKPOT_PROBABILITY,
        softPity: DEFAULT_BANME_SOFT_PITY,
        hardPity: DEFAULT_BANME_HARD_PITY,
        upDuration: DEFAULT_BANME_UP_DURATION,
        loseDuration: DEFAULT_BANME_LOSE_DURATION,
      },
    },
    openai: { enabled: true },
  }
}
