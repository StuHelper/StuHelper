import type { Context, Session } from 'koishi'
import type { PlatformClient } from '@stuhelper/koishi-shared'

import { createKickMemberBlacklist } from './member-blacklist-backend'

interface BlackKickHost {
  readonly ctx: Context
  readonly memberBlacklistBackend?: Pick<PlatformClient, 'createMemberBlacklist'>
  logCommand(session: Session, command: string, target: string, result: string, success?: boolean): void
}

interface BlackKickInput {
  readonly userId: string
  readonly targetGroup: string
}

export async function handleBlackKick(
  host: BlackKickHost,
  session: Session,
  input: BlackKickInput,
): Promise<void> {
  await createKickMemberBlacklist(requireMemberBlacklistBackend(host), {
    platform: session.platform,
    guildID: input.targetGroup,
    subjectID: input.userId,
    operatorQQID: session.userId,
    rawCommand: session.content?.trim() || '',
  })
  host.logCommand(session, 'kick', input.userId, `成功：移出群聊并加入黑名单：${input.targetGroup}`)
  await host.ctx.stuhelperGroupCenter.pushMessage(
    session.bot,
    `[黑名单] 用户 ${input.userId} 被踢出群 ${input.targetGroup} 并加入本群黑名单`,
    'blacklist',
  )
}

function requireMemberBlacklistBackend(
  host: BlackKickHost,
): Pick<PlatformClient, 'createMemberBlacklist'> {
  if (!host.memberBlacklistBackend) {
    throw new Error('member blacklist backend client is required for kick blacklist command')
  }
  return host.memberBlacklistBackend
}
