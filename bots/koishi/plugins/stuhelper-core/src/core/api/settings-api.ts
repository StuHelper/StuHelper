import type { PluginSettings } from '../settings'
import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

type PlainRecord = Record<string, unknown>
type PublicSettings = Omit<PluginSettings, 'openai'> & {
  openai: Omit<PluginSettings['openai'], 'apiKey'> & {
    apiKeyConfigured: boolean
    apiKeyMasked: string
  }
}

export function registerSettingsAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/settings/get', async function () {
    return handleSettingsGet(api, this)
  })
  api.addAuthorityListener(
    'stuhelperGroupCenter/settings/update',
    async function (params?: { settings?: unknown }) {
      return handleSettingsUpdate(api, this, params?.settings)
    },
  )
  api.addAuthorityListener('stuhelperGroupCenter/settings/reset', async function () {
    return handleSettingsReset(api, this)
  })
}

async function handleSettingsGet(api: WebSocketAPIContext, client: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'settings')
    return success(toConsoleSettings(api.service.settings.settings))
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '获取设置失败')
  }
}

async function handleSettingsUpdate(api: WebSocketAPIContext, client: unknown, settings: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'settings')
    if (!isSettingsPatch(settings)) {
      return error('无效的设置数据')
    }
    const normalizedSettings = normalizeSettingsPatch(settings)
    await api.service.settings.update(normalizedSettings)
    api.ctx.logger('stuhelperGroupCenter').info('设置已更新')
    return success({ success: true })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('更新设置失败:', cause)
    return error(cause instanceof Error ? cause.message : '更新设置失败')
  }
}

function isSettingsPatch(value: unknown): value is Partial<PluginSettings> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isPlainRecord(value: unknown): value is PlainRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function toConsoleSettings(settings: PluginSettings): PublicSettings {
  const snapshot = structuredClone(settings)
  const { apiKey, ...openai } = snapshot.openai

  return {
    ...snapshot,
    openai: {
      ...openai,
      apiKeyConfigured: apiKey.length > 0,
      apiKeyMasked: maskSecret(apiKey),
    },
  }
}

function normalizeSettingsPatch(settings: Partial<PluginSettings>): Partial<PluginSettings> {
  const patch = { ...(settings as unknown as PlainRecord) }
  if (!isPlainRecord(patch.openai)) {
    return patch as Partial<PluginSettings>
  }

  const openai = { ...patch.openai }
  const clearApiKey = openai.clearApiKey === true
  const newApiKey = typeof openai.newApiKey === 'string' ? openai.newApiKey.trim() : ''
  const legacyApiKey = typeof openai.apiKey === 'string' ? openai.apiKey.trim() : ''
  const replacementApiKey = newApiKey || legacyApiKey

  delete openai.apiKeyConfigured
  delete openai.apiKeyMasked
  delete openai.newApiKey
  delete openai.clearApiKey
  delete openai.apiKey

  if (clearApiKey && replacementApiKey) {
    throw new Error('不能同时清除和替换 API Key')
  }
  if (clearApiKey) {
    openai.apiKey = ''
  } else if (replacementApiKey) {
    openai.apiKey = replacementApiKey
  }

  patch.openai = openai
  return patch as Partial<PluginSettings>
}

function maskSecret(value: string): string {
  if (!value) return ''
  if (value.length <= 8) return '已配置'
  return `${value.slice(0, 4)}...${value.slice(-4)}`
}

async function handleSettingsReset(api: WebSocketAPIContext, client: unknown) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertGlobalConsoleScope(scope, 'settings')
    await api.service.settings.reset()
    api.ctx.logger('stuhelperGroupCenter').info('设置已重置为默认值')
    return success({ success: true })
  } catch (cause) {
    api.ctx.logger('stuhelperGroupCenter').error('重置设置失败:', cause)
    return error(cause instanceof Error ? cause.message : '重置设置失败')
  }
}
