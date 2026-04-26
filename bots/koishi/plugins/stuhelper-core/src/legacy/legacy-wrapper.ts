import { Context, Schema } from 'koishi'

import {
  createCoreConfigSchema,
  type StuhelperCoreConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-core'

export type Config = StuhelperCoreConfig

export const Config: Schema<Config> = createCoreConfigSchema()

export function applyLegacyFeatures(_ctx: Context, _config: Config) {}

export default {
  name,
  Config,
  apply: applyLegacyFeatures,
}
