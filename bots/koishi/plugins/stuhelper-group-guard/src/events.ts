import type { Context } from 'koishi'
import type { AdmissionRuntimeSettingsStore } from '@stuhelper/koishi-shared'

import type { MemberGuardService } from './member-guard'
import type { MessageGuardService } from './message-guard'
import type { GuardBotRuntime } from './member-guard'

interface EventLogger {
  error(message: string, ...args: unknown[]): void
}

interface EventDeps {
  memberGuard: MemberGuardService
  messageGuard?: MessageGuardService
  runtimeSettings?: AdmissionRuntimeSettingsStore
  logger: EventLogger
  scanIntervalSeconds: number
}

export function registerGroupGuardEvents(ctx: Context, deps: EventDeps) {
  ctx.on('guild-member-added', (session) => {
    return deps.memberGuard.handleGuildMemberAdded(session)
  })

  ctx.on('guild-member-request', (session) => {
    return deps.memberGuard.handleGuildMemberRequest(session)
  })

  if (deps.messageGuard) {
    ctx.on('message', (session) => {
      if (deps.runtimeSettings) {
        return deps.runtimeSettings.isModerationEnabled().then((enabled) => {
          if (!enabled) return
          return deps.messageGuard!.handleMessage(session)
        })
      }
      return deps.messageGuard!.handleMessage(session)
    })

    ctx.on('message-deleted', (session) => {
      if (deps.runtimeSettings) {
        return deps.runtimeSettings.isModerationEnabled().then((enabled) => {
          if (!enabled) return
          return deps.messageGuard!.handleMessageDeleted(session)
        })
      }
      return deps.messageGuard!.handleMessageDeleted(session)
    })
  }

  ctx.setInterval(() => {
    return Promise.resolve(deps.runtimeSettings?.isFallbackScanEnabled() ?? true)
      .then((enabled) => {
        if (!enabled) return
        return deps.memberGuard
          .scanPendingMembers(ctx.bots as GuardBotRuntime[])
          .catch((error) => deps.logger.error('group guard scheduled scan failed', error))
      })
      .catch((error) => deps.logger.error('group guard scheduled scan readiness failed', error))
  }, deps.scanIntervalSeconds * 1000)
}
