import type { Session } from 'koishi'

import {
  canExecuteCommand,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'

export async function ensureAdminCommandAccess(
  store: ModerationStore,
  session: Session | undefined,
  commandId: string,
  targetGuildId = session?.guildId,
) {
  const guildId = targetGuildId
  if (!session || !guildId) return
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    store.getMemberRoles(guildId, session.userId),
  ])
  const allowed = canExecuteCommand({
    authority: resolveAuthority(session),
    memberRoles,
    policy,
  })
  return allowed ? undefined : '命令权限不足。'
}

export function resolveGuildId(session: Session | undefined, guildId: string | undefined) {
  return guildId?.trim() || session?.guildId || ''
}

function resolveAuthority(session: Session | undefined) {
  const target = session as { user?: { authority?: number } } | undefined
  return target?.user?.authority ?? 0
}
