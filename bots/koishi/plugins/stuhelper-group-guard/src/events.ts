import type { Context } from 'koishi'

import type { MemberGuardService } from './member-guard'
import type { MessageGuardService } from './message-guard'
import type { GuardBotRuntime } from './member-guard'

interface EventLogger {
  error(message: string, ...args: unknown[]): void
}

interface EventDeps {
  memberGuard: MemberGuardService
  messageGuard?: MessageGuardService
  logger: EventLogger
  scanIntervalSeconds: number
}

export function registerGroupGuardEvents(ctx: Context, deps: EventDeps) {
  ctx.on('guild-member-added', (session) => {
    return deps.memberGuard.handleGuildMemberAdded(session)
  })

  if (deps.messageGuard) {
    ctx.on('message', (session) => {
      return deps.messageGuard!.handleMessage(session)
    })

    ctx.on('message-deleted', (session) => {
      return deps.messageGuard!.handleMessageDeleted(session)
    })
  }

  ctx.setInterval(() => {
    return deps.memberGuard
      .scanPendingMembers(ctx.bots as GuardBotRuntime[])
      .catch((error) => deps.logger.error('group guard scheduled scan failed', error))
  }, deps.scanIntervalSeconds * 1000)
}
