import type { PluginSettings } from '../settings'
import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import { assertGlobalConsoleScope } from './console-guild-scope'

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
    return success(api.service.settings.settings)
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
    await api.service.settings.update(settings)
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
