import type { Context } from 'koishi'

import type { RuntimeCommand, RuntimeCommandDef, RuntimeModuleMeta } from './types'

export function registerRuntimeCommand(
  ctx: Context,
  meta: RuntimeModuleMeta,
  def: RuntimeCommandDef,
): RuntimeCommand {
  const cmdDef = def.args ? `${def.name} ${def.args}` : def.name
  const permNode = def.permNode || def.name.replace(/\./g, '-')
  const permId = `${meta.name}.${permNode}`
  const permDesc = def.permDesc || def.desc

  if (!def.skipAuth) {
    ctx.stuhelperGroupCenter.auth.registerPermission({
      id: permId,
      name: def.desc,
      description: permDesc,
      group: meta.description,
    })
  }

  ctx.stuhelperGroupCenter.auth.registerCommand({
    name: def.name,
    desc: def.desc,
    args: def.args,
    usage: def.usage,
    examples: def.examples,
    module: meta.name,
    moduleDesc: meta.description,
    permId: def.skipAuth ? undefined : permId,
    skipAuth: def.skipAuth,
  })

  const command = ctx.command(cmdDef, def.desc)
  if (!def.skipAuth) {
    command.before(async ({ session }) => {
      if (!session) return
      if (!ctx.stuhelperGroupCenter.auth.check(session, permId)) {
        return '你没有权限执行此操作喵...'
      }
    })
  }

  return command
}
