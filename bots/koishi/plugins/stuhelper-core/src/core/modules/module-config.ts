import type { Context } from 'koishi'

import type { Config } from '../../types'

export function getRequiredPluginConfig(ctx: Context): Config {
  const service = ctx.stuhelperGroupCenter
  if (!service) {
    throw new Error('stuhelperGroupCenter service is required before reading runtime module config')
  }
  return service.pluginConfig
}
