import type { Context, Logger } from 'koishi'

const LOGGER_NAMESPACE = 'stuhelper'

export function createPluginLogger(ctx: Context, scope: string): Logger {
  return ctx.logger(`${LOGGER_NAMESPACE}:${scope}`)
}
