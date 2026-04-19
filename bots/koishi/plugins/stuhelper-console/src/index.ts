import { resolve } from 'node:path'

import { Context, Schema } from 'koishi'

import {
  createConsolePluginConfigSchema,
  registerGuardMemberModel,
  registerGuardPolicyModels,
  type StuhelperConsolePluginConfig,
} from '@stuhelper/koishi-shared'
import { registerModerationModels } from '@stuhelper/koishi-moderation-core'

import { registerConsoleListeners } from './controller'
import type {
  StuhelperCommandPolicyInput,
  StuhelperGuardBatchActionInput,
  StuhelperGuardBindingInput,
  StuhelperGuardTemplateInput,
  StuhelperConsoleKeywordRule,
  StuhelperMemberRoleInput,
  StuhelperReviewActionInput,
} from './console-types'
import { StuhelperConsoleDataService } from './dashboard-service'

declare module '@koishijs/console' {
  interface Events {
    'stuhelper-console/refresh'(): void | string | Promise<void | string>
    'stuhelper-console/guard-action'(input: StuhelperGuardBatchActionInput): void | string | Promise<void | string>
    'stuhelper-console/review-action'(input: StuhelperReviewActionInput): void | string | Promise<void | string>
    'stuhelper-console/save-keyword-rule'(input: StuhelperConsoleKeywordRule): void | string | Promise<void | string>
    'stuhelper-console/save-member-roles'(input: StuhelperMemberRoleInput): void | string | Promise<void | string>
    'stuhelper-console/save-command-policy'(input: StuhelperCommandPolicyInput): void | string | Promise<void | string>
    'stuhelper-console/save-guard-template'(input: StuhelperGuardTemplateInput): void | string | Promise<void | string>
    'stuhelper-console/save-guard-binding'(input: StuhelperGuardBindingInput): void | string | Promise<void | string>
  }
}

export const name = 'stuhelper-console'
export const inject = ['console', 'database']

export type Config = StuhelperConsolePluginConfig

export const Config: Schema<Config> = createConsolePluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  if (!config.console.enabled) {
    return
  }

  registerGuardMemberModel(ctx)
  registerGuardPolicyModels(ctx)
  registerModerationModels(ctx)

  ctx.console.addEntry({
    dev: resolve(__dirname, '../client/index.ts'),
    prod: resolve(__dirname, '../dist'),
  })

  ctx.plugin(StuhelperConsoleDataService, {
    title: config.console.title,
  })
  registerConsoleListeners(ctx)
}

export default {
  name,
  inject,
  Config,
  apply,
}
