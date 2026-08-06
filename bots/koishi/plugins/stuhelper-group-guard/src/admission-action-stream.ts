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
import { opaqueLogReference } from './log-reference'

interface AdmissionActionStreamLogger {
  info(message: string, ...args: unknown[]): void
  warn(message: string, ...args: unknown[]): void
}

interface AdmissionActionStreamDeps {
  platform: PlatformClient
  memberGuard: MemberGuardService
  logger: AdmissionActionStreamLogger
  config?: StuhelperAdmissionActionStreamConfig
  isEnabled?: () => boolean | Promise<boolean>
}

const DEFAULT_RECONNECT_DELAY_SECONDS = 5
const MAX_RECONNECT_DELAY_MS = 5 * 60_000
const STABLE_CONNECTION_RESET_MS = 60_000

export interface AdmissionActionStreamController {
  refresh(): Promise<void>
  close(): void
}

export function registerAdmissionActionStreams(ctx: Context, deps: AdmissionActionStreamDeps): AdmissionActionStreamController {
  const streams = new AdmissionActionStreamRuntime(ctx, deps)
  ctx.on('ready', () => {
    return streams.refresh()
  })
  ctx.on('dispose', () => streams.close())
  return streams
}

class AdmissionActionStreamRuntime {
  private readonly handles = new Map<string, { close(): void }>()
  private readonly reconnectTimers = new Map<string, ReturnType<typeof setTimeout>>()
  private readonly stabilityTimers = new Map<string, ReturnType<typeof setTimeout>>()
  private readonly reconnectAttempts = new Map<string, number>()
  private disposed = false

  constructor(
    private readonly ctx: Context,
    private readonly deps: AdmissionActionStreamDeps,
  ) {}

  async refresh() {
    if (this.disposed) {
      return
    }
    let enabled: boolean
    try {
      enabled = await this.readEnabled()
    } catch (error) {
      this.closeActiveStreams()
      for (const bot of this.ctx.bots as GuardBotRuntime[]) {
        this.scheduleReconnectForBot(bot, error, 'runtime-setting')
      }
      return
    }
    if (!enabled) {
      this.closeActiveStreams()
      return
    }
    for (const bot of this.ctx.bots as GuardBotRuntime[]) {
      this.ensureBotStream(bot)
    }
  }

  close() {
    this.disposed = true
    this.closeActiveStreams()
  }

  private closeActiveStreams() {
    for (const timer of this.reconnectTimers.values()) {
      clearTimeout(timer)
    }
    this.reconnectTimers.clear()
    for (const timer of this.stabilityTimers.values()) {
      clearTimeout(timer)
    }
    this.stabilityTimers.clear()
    for (const handle of this.handles.values()) {
      handle.close()
    }
    this.handles.clear()
    this.reconnectAttempts.clear()
  }

  private async readEnabled() {
    return await (this.deps.isEnabled?.() ?? true)
  }

  private ensureBotStream(bot: GuardBotRuntime) {
    if (this.disposed || !isAdmissionActionPlatform(bot)) {
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
      onOpen: () => this.handleStreamOpen(key, bot, platform),
      onAction: (action) => this.handleAction(key, bot, action),
      onError: (error) => this.handleStreamError(key, bot, error),
    })
    this.handles.set(key, handle)
  }

  private handleStreamOpen(key: string, bot: GuardBotRuntime, platform: string) {
    const reconnectAttempt = this.reconnectAttempts.get(key)
    if (typeof reconnectAttempt === 'undefined') {
      this.deps.logger.info('admission action stream connected', {
        platform,
        streamRef: opaqueLogReference('bot', bot.selfId),
      })
      return
    }
    this.clearStabilityTimer(key)
    const timer = setTimeout(() => {
      this.stabilityTimers.delete(key)
      this.reconnectAttempts.delete(key)
    }, STABLE_CONNECTION_RESET_MS)
    this.stabilityTimers.set(key, timer)
  }

  private async handleAction(key: string, bot: GuardBotRuntime, action: AdmissionPendingAction) {
    await this.deps.memberGuard.handleQueuedAdmissionAction(bot, action)
  }

  private handleStreamError(key: string, bot: GuardBotRuntime, error: unknown) {
    const handle = this.handles.get(key)
    if (!handle) {
      return
    }
    this.handles.delete(key)
    handle.close()
    this.clearStabilityTimer(key)
    this.scheduleReconnect(key, bot, error, 'stream')
  }

  private scheduleReconnectForBot(
    bot: GuardBotRuntime,
    error: unknown,
    source: 'runtime-setting' | 'stream',
  ) {
    if (this.disposed || !isAdmissionActionPlatform(bot)) {
      return
    }
    const key = `${requireAdmissionActionPlatform(bot)}:${bot.selfId}`
    this.scheduleReconnect(key, bot, error, source)
  }

  private scheduleReconnect(
    key: string,
    bot: GuardBotRuntime,
    error: unknown,
    source: 'runtime-setting' | 'stream',
  ) {
    if (this.disposed || this.reconnectTimers.has(key)) {
      return
    }
    const attempt = this.reconnectAttempts.get(key) ?? 0
    const delayMs = admissionActionReconnectDelayMs(
      this.deps.config?.reconnectDelaySeconds,
      attempt,
    )
    this.reconnectAttempts.set(key, attempt + 1)
    if (shouldLogReconnectAttempt(attempt + 1)) {
      this.deps.logger.warn(
        source === 'stream'
          ? 'admission action stream disconnected; reconnect scheduled'
          : 'admission action stream runtime setting unavailable; reconnect scheduled',
        {
          streamRef: opaqueLogReference('bot', bot.selfId),
          reconnectAttempt: attempt + 1,
          reconnectDelayMs: delayMs,
          error: error instanceof Error ? error.message : String(error),
        },
      )
    }
    const timer = setTimeout(async () => {
      this.reconnectTimers.delete(key)
      if (this.disposed) {
        return
      }
      let enabled: boolean
      try {
        enabled = await this.readEnabled()
      } catch (settingError) {
        this.scheduleReconnect(key, bot, settingError, 'runtime-setting')
        return
      }
      if (!enabled) {
        this.clearReconnectState(key)
        return
      }
      this.openBotStream(key, bot)
    }, delayMs)
    this.reconnectTimers.set(key, timer)
  }

  private clearStabilityTimer(key: string) {
    const timer = this.stabilityTimers.get(key)
    if (!timer) {
      return
    }
    clearTimeout(timer)
    this.stabilityTimers.delete(key)
  }

  private clearReconnectState(key: string) {
    this.clearStabilityTimer(key)
    this.reconnectAttempts.delete(key)
  }
}

export function shouldLogReconnectAttempt(attempt: number): boolean {
  return attempt === 1 || (attempt > 1 && (attempt & (attempt - 1)) === 0)
}

export function admissionActionReconnectDelayMs(
  configuredBaseSeconds: number | undefined,
  attempt: number,
  random: () => number = Math.random,
) {
  const seconds = Number.isFinite(configuredBaseSeconds)
    ? configuredBaseSeconds as number
    : DEFAULT_RECONNECT_DELAY_SECONDS
  const baseMs = Math.min(
    MAX_RECONNECT_DELAY_MS,
    Math.max(1, seconds) * 1000,
  )
  const exponent = Math.min(16, Math.max(0, Math.floor(attempt)))
  const exponentialMs = Math.min(
    MAX_RECONNECT_DELAY_MS,
    baseMs * (2 ** exponent),
  )
  const randomValue = Math.min(1, Math.max(0, random()))
  const jitteredMs = exponentialMs * (0.8 + randomValue * 0.4)
  return Math.round(Math.min(
    MAX_RECONNECT_DELAY_MS,
    Math.max(baseMs, jitteredMs),
  ))
}
