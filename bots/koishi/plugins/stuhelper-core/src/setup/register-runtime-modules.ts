import { Context, Logger } from 'koishi'

import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

const logger = new Logger('stuhelper-core')

export function registerRuntimeModules(_ctx: Context, _config?: Config) {
  logger.info('stuhelper-core 旧群管运行时模块已停用，仅注册 WebUI 与 Console API')
}
