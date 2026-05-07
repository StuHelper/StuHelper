import type { Context } from 'koishi'

import type { StuhelperGroupCenterService } from '../services/stuhelper-group-center.service'
import { registerAuthAPI } from './auth-api'
import { registerChatAPI } from './chat-api'
import { registerConfigAPI } from './config-api'
import { registerOpsAPI } from './ops-api'
import { registerStatsAPI } from './stats-api'
import { registerSubscriptionsAPI } from './subscriptions-api'
import { registerWarnsAPI } from './warns-api'
import { createWebSocketAPIContext } from './websocket-api-context'
import type { MemberBlacklistStatsBackend } from './dashboard-blacklist-stats'

export { registerMemberBlacklistConsoleAPI } from './member-blacklist-console-api'

export function registerWebSocketAPI(
  ctx: Context,
  service: StuhelperGroupCenterService,
  memberBlacklistBackend?: MemberBlacklistStatsBackend,
) {
  if (!ctx.console) {
    ctx.logger('stuhelperGroupCenter').warn('console 服务未启用，WebSocket API 跳过注册')
    return
  }

  const api = createWebSocketAPIContext(ctx, service)
  registerConfigAPI(api)
  registerAuthAPI(api)
  registerWarnsAPI(api)
  registerSubscriptionsAPI(api)
  registerStatsAPI(api, memberBlacklistBackend)
  registerOpsAPI(api)
  registerChatAPI(api)
}

export * from './page-api'
export * from './review-actions'
export * from './governance-actions'
