declare module 'koishi-plugin-stuhelper-group-guard' {
  import type { Context } from 'koishi'
  import type { StuhelperGroupGuardPluginConfig } from '@stuhelper/koishi-shared'

  export function registerGroupGuardRuntimeModels(ctx: Context): void

  export function startGroupGuardRuntime(
    ctx: Context,
    config: StuhelperGroupGuardPluginConfig,
  ): void
}
