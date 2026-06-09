import type { Session } from 'koishi'

import {
  canExecuteCommand,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
  type AdmissionRuntimeSettingsStore,
  renderMessageTemplate,
  resolveAdminMessages,
} from '@stuhelper/koishi-shared'

type AdminMessages = ReturnType<typeof resolveAdminMessages>

export async function ensureAdminCommandAccess(input: {
  readonly store: ModerationStore
  readonly session: Session | undefined
  readonly commandId: string
  readonly targetGuildId?: string
  readonly runtimeSettings?: AdmissionRuntimeSettingsStore
  readonly messages?: AdminMessages
}) {
  const { store, session, commandId } = input
  if (input.runtimeSettings && !await input.runtimeSettings.isAdminCommandsEnabled()) {
    return renderMessageTemplate(resolveAdminMessages(input.messages).adminCommandsDisabled)
  }
  const targetGuildId = input.targetGuildId ?? session?.guildId
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
  return allowed
    ? undefined
    : renderMessageTemplate(resolveAdminMessages(input.messages).commandAccessDenied)
}

export function resolveGuildId(session: Session | undefined, guildId: string | undefined) {
  return guildId?.trim() || session?.guildId || ''
}

function resolveAuthority(session: Session | undefined) {
  const target = session as { user?: { authority?: number } } | undefined
  return target?.user?.authority ?? 0
}
