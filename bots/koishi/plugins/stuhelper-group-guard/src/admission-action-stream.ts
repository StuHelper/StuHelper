import type { Context } from 'koishi'

import type {
  AdmissionPendingAction,
  PlatformClient,
  StuhelperAdmissionActionStreamConfig,
} from '@stuhelper/koishi-shared'

import {
  isAdmissionActionPlatform,
  requireAdmissionActionPlatform,
} from './admission-action-boundary'
import type { GuardBotRuntime, MemberGuardService } from './member-guard'

interface AdmissionActionStreamLogger {
  info(message: string, ...args: unknown[]): void
  warn(message: string, ...args: unknown[]): void
}

interface AdmissionActionStreamDeps {
  platform: PlatformClient
  memberGuard: MemberGuardService
  logger: AdmissionActionStreamLogger
  config?: StuhelperAdmissionActionStreamConfig
}

export function registerAdmissionActionStreams(ctx: Context, deps: AdmissionActionStreamDeps) {
  if (deps.config?.enabled === false) {
    return
  }
  const streams = new AdmissionActionStreamRuntime(ctx, deps)
  ctx.on('ready', () => streams.start())
  ctx.on('dispose', () => streams.close())
}

class AdmissionActionStreamRuntime {
  private readonly handles = new Map<string, { close(): void }>()
  private readonly reconnectTimers = new Map<string, ReturnType<typeof setTimeout>>()
  private closed = false

  constructor(
    private readonly ctx: Context,
    private readonly deps: AdmissionActionStreamDeps,
  ) {}

  start() {
    for (const bot of this.ctx.bots as GuardBotRuntime[]) {
      this.ensureBotStream(bot)
    }
  }

  close() {
    this.closed = true
    for (const timer of this.reconnectTimers.values()) {
      clearTimeout(timer)
    }
    this.reconnectTimers.clear()
    for (const handle of this.handles.values()) {
      handle.close()
    }
    this.handles.clear()
  }

  private ensureBotStream(bot: GuardBotRuntime) {
    if (this.closed || !isAdmissionActionPlatform(bot)) {
      return
    }
    const key = `${requireAdmissionActionPlatform(bot)}:${bot.selfId}`
    if (this.handles.has(key) || this.reconnectTimers.has(key)) {
      return
    }
    this.openBotStream(key, bot)
  }

  private openBotStream(key: string, bot: GuardBotRuntime) {
    const platform = requireAdmissionActionPlatform(bot)
    const handle = this.deps.platform.streamAdmissionActions({
      platform,
      botSelfID: bot.selfId,
      limit: 50,
    }, {
      onOpen: () => this.deps.logger.info('admission action stream connected', { platform, botSelfID: bot.selfId }),
      onAction: (action) => this.handleAction(bot, action),
      onError: (error) => this.handleStreamError(key, bot, error),
    })
    this.handles.set(key, handle)
  }

  private async handleAction(bot: GuardBotRuntime, action: AdmissionPendingAction) {
    await this.deps.memberGuard.handleQueuedAdmissionAction(bot, action)
  }

  private handleStreamError(key: string, bot: GuardBotRuntime, error: unknown) {
    this.handles.delete(key)
    this.deps.logger.warn('admission action stream disconnected; reconnect scheduled', {
      botSelfID: bot.selfId,
      error: error instanceof Error ? error.message : String(error),
    })
    const delayMs = Math.max(1, this.deps.config?.reconnectDelaySeconds ?? 5) * 1000
    const timer = setTimeout(() => {
      this.reconnectTimers.delete(key)
      this.openBotStream(key, bot)
    }, delayMs)
    this.reconnectTimers.set(key, timer)
  }
}
