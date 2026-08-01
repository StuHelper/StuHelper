import { build } from '@koishijs/client/lib'
import { resolve } from 'node:path'

// @koishijs/client 5.30 still registers commands through Yakumo 1's
// ctx.register() API. Keep the upstream build implementation, but expose it
// through Yakumo 3's CLI service until the package publishes a native adapter.
export const inject = ['cli', 'yakumo']

export function apply(ctx) {
  ctx.cli.command('client [...packages]', 'Build Koishi client extensions').action(async ({ args }) => {
    await ctx.yakumo.initialize()
    const paths = ctx.yakumo.locate(args)

    for (const path of paths) {
      const meta = ctx.yakumo.workspaces[path]
      const dependencies = {
        ...meta.dependencies,
        ...meta.devDependencies,
        ...meta.peerDependencies,
        ...meta.optionalDependencies,
      }

      let config = {}
      if (meta.yakumo?.client) {
        const filename = resolve(ctx.yakumo.cwd + path, meta.yakumo.client)
        const module = await import(filename)
        if (typeof module.default === 'function') {
          await module.default()
          continue
        }
        config = module.default
      } else if (!dependencies['@koishijs/client']) {
        continue
      }

      await build(ctx.yakumo.cwd + path, config)
    }
  })
}
