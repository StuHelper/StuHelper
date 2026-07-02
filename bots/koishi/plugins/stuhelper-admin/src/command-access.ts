import type { Session } from 'koishi'

import {
  canExecuteCommand,
  createFallbackCommandPolicy,
  type ModerationStore,
} from '@stuhelper/koishi-moderation-core'
import {
  type AdmissionRuntimeSettingsStore,
  renderMessageTemplate,
  resolveAdminMessages,
} from '@stuhelper/koishi-shared'

type AdminMessages = ReturnType<typeof resolveAdminMessages>

/**
 * 无策略记录时的兜底最低权限，与 group-guard 侧
 * DEFAULT_ADMISSION_COMMAND_POLICY 的 fail-closed 语义保持一致。
 */
const FALLBACK_ADMIN_COMMAND_AUTHORITY = 4

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
  if (!session) {
    return renderMessageTemplate(resolveAdminMessages(input.messages).commandAccessDenied)
  }
  const guildId = (input.targetGuildId ?? session.guildId ?? '').trim()
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    guildId ? store.getMemberRoles(guildId, session.userId) : Promise.resolve<string[]>([]),
  ])
  const allowed = canExecuteCommand({
    authority: resolveAuthority(session),
    memberRoles,
    policy: policy ?? createFallbackCommandPolicy(commandId, FALLBACK_ADMIN_COMMAND_AUTHORITY),
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
