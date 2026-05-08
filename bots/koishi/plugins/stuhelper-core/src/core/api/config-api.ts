import { error, success } from './api-response'
import type { WebSocketAPIContext } from './api-context'
import { assertConsoleGuildAccess } from './console-guild-scope'
import { filterGuildEntries } from './scope-filters'

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

export function registerConfigAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/config/reload', async () => handleConfigReload(api))
  api.addAuthorityListener('stuhelperGroupCenter/config/list', async function (params?: { fetchNames?: boolean }) {
    const scope = await api.resolveConsoleScope(this)
    return success(buildConfigList(api, filterGuildEntries(api.service.data.groupConfig.getAll(), scope), params))
  })
  api.addAuthorityListener('stuhelperGroupCenter/config/get', async function (params: { guildId: string }) {
    return handleConfigGet(api, this, params.guildId)
  })
  api.addAuthorityListener('stuhelperGroupCenter/config/update', async function (params: { guildId: string, config: any }) {
    const scope = await api.resolveConsoleScope(this)
    assertConsoleGuildAccess(scope, params.guildId, 'group config')
    api.service.data.groupConfig.set(params.guildId, params.config)
    await api.service.data.groupConfig.flush()
    return success({ success: true })
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
  scopedConfigs: [string, any][],
  params?: { fetchNames?: boolean },
) {
  const results: Record<string, any> = {}
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
