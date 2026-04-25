import { Context, Logger } from 'koishi'
import type {} from '@koishijs/plugin-console'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import { resolveBrowserEntry } from '../browser-entry'

const logger = new Logger('stuhelper-core')

export function registerConsoleEntry(ctx: Context, _config?: Config) {
  ctx.inject(['console'], (consoleCtx) => {
    consoleCtx.console.addEntry(resolveBrowserEntry())
    logger.info('StuHelper 群管中心控制台入口已注册')
  })
}
