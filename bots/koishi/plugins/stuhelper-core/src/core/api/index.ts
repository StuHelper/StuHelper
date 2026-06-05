import { createPlatformClient, type StuhelperPlatformConfig } from '@stuhelper/koishi-shared'
import type { Context } from 'koishi'
import type {} from '@koishijs/plugin-console'

import { StuhelperGroupCenterService } from '../services/stuhelper-group-center.service'
import { createAuthority4ListenerRegistrar } from './authority-listener'
import { registerAuthAPI } from './auth-api'
import { registerCacheAPI } from './cache-api'
import { registerChatAPI } from './chat-api'
import { createChatImageAccessRegistry } from './chat-image-fetch'
import { registerConfigAPI } from './config-api'
import { registerLogsAPI } from './logs-api'
import { registerMemberBlacklistConsoleAPI } from './member-blacklist-console-api'
import { registerSettingsAPI } from './settings-api'
import { registerStatsAPI } from './stats-api'
import { registerSubscriptionsAPI } from './subscriptions-api'
import { registerWarnsAPI } from './warns-api'
import { resolveRequiredConsoleGuildScope } from './console-guild-scope'
import type { WebSocketAPIContext } from './api-context'

const pkg = require('../../../package.json')

export function registerWebSocketAPI(
  ctx: Context,
  service: StuhelperGroupCenterService,
  platformConfig: StuhelperPlatformConfig,
) {
  if (!ctx.console) {
    ctx.logger('stuhelperGroupCenter').warn('console 服务未启用，WebSocket API 跳过注册')
    return
  }

  const addAuthorityListener = createAuthority4ListenerRegistrar(ctx.console)
  const platform = createPlatformClient(platformConfig)
  const apiContext: WebSocketAPIContext = {
    ctx,
    service,
    addAuthorityListener,
    platform,
    packageVersion: pkg.version,
    resolveConsoleScope: (client: unknown) => resolveRequiredConsoleGuildScope(client, {
      roles: service.auth.getRoles(),
      getUserRoleIds: (userId: string) => service.auth.getUserRoleIds(userId),
      listBindingsByAuthId: (authId: number) => ctx.database.get('binding', { aid: authId }),
    }),
  }

  registerMemberBlacklistConsoleAPI(ctx, service, platform)
  registerConfigAPI(apiContext)
  registerAuthAPI(apiContext)
  registerWarnsAPI(apiContext)
  registerSubscriptionsAPI(apiContext)
  registerStatsAPI(apiContext)
  registerLogsAPI(apiContext)
  registerSettingsAPI(apiContext)
  registerCacheAPI(apiContext)
  registerChatAPI(apiContext, { imageAccess: createChatImageAccessRegistry() })
}

export * from './page-api'
export * from './review-actions'
export * from './governance-actions'
