import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import {
  registerGovernanceActionAPI,
  registerPageAPI,
  registerReviewActionAPI,
  registerWebSocketAPI,
} from '../core'
import { validateConsoleAdminPassword } from '../console-auth'

const logger = new Logger('stuhelper-core')

export function registerConsoleApi(ctx: Context, config?: Config) {
  const coreConfig = requireConfig(config)

  ctx.inject(['console', 'database', 'stuhelperGroupCenter', 'auth'], (apiCtx) => {
    validateConsoleAdminPassword(process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD)
    registerWebSocketAPI(apiCtx, apiCtx.stuhelperGroupCenter, coreConfig.platform)
    registerPageAPI(apiCtx, {
      service: apiCtx.stuhelperGroupCenter,
      platform: coreConfig.platform,
      guard: coreConfig.guard,
    })
    registerReviewActionAPI(apiCtx)
    registerGovernanceActionAPI(apiCtx)
    logger.info('WebSocket API registered')
  })
}

function requireConfig(config: Config | undefined) {
  if (!config) {
    throw new Error('stuhelper-core config is required to register console API')
  }
  return config
}
