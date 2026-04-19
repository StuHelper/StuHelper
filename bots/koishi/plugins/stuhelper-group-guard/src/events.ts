import type { Context } from 'koishi'

import type { MemberGuardService } from './member-guard'
import type { MessageGuardService } from './message-guard'
import type { GuardBotRuntime } from './member-guard'

interface EventDeps {
  memberGuard: MemberGuardService
  messageGuard: MessageGuardService
  scanIntervalSeconds: number
}

export function registerGroupGuardEvents(ctx: Context, deps: EventDeps) {
  ctx.on('guild-member-added', (session) => {
    void deps.memberGuard.handleGuildMemberAdded(session)
  })

  ctx.on('message', (session) => {
    void deps.messageGuard.handleMessage(session)
  })

  ctx.on('message-deleted', (session) => {
    void deps.messageGuard.handleMessageDeleted(session)
  })

  ctx.setInterval(() => {
    void deps.memberGuard.scanPendingMembers(ctx.bots as GuardBotRuntime[])
  }, deps.scanIntervalSeconds * 1000)
}
