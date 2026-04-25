import { Context } from 'koishi'

import type { ModerationBot, ModerationRuntimeRef } from '@stuhelper/koishi-moderation-core'

export type ConsoleManagedBot = ModerationBot

export function resolveManagedBot(ctx: Context, runtime: ModerationRuntimeRef): ConsoleManagedBot {
  const bot = ctx.bots.find((item) => item.platform === runtime.platform && item.selfId === runtime.botSelfId)
  if (!bot) {
    throw new Error(`console bot not found: ${runtime.platform}:${runtime.botSelfId}`)
  }
  return bot as unknown as ConsoleManagedBot
}
