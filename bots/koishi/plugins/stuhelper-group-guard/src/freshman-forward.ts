import { h, type Universal } from 'koishi'

import type { FreshmanForwardItem } from '@stuhelper/koishi-shared'

import { formatFreshmanForwardSummary } from './admission-format'

export function resolveFreshmanForwardBot(
  bots: readonly FreshmanForwardBot[],
  item: FreshmanForwardItem,
) {
  if (item.botSelfID) {
    const bot = bots.find((runtime) => botMatchesForward(runtime, item))
    if (!bot) throw new Error(`freshman forward bot not found: ${item.platform || '*'}:${item.botSelfID}`)
    return bot
  }
  if (bots.length === 1) return bots[0]
  throw new Error(`freshman forward ${item.application.id} missing botSelfID for multi-bot runtime`)
}

export async function forwardFreshmanMaterial(
  bot: Universal.Methods,
  item: FreshmanForwardItem,
) {
  if (!item.materialURL) {
    throw new Error(`freshman forward ${item.application.id} missing materialURL`)
  }
  if (!item.managementGuildIDs.length) {
    throw new Error(`freshman forward ${item.application.id} missing managementGuildIDs`)
  }
  const content = [
    h.image(item.materialURL),
    formatFreshmanForwardSummary(item),
  ].join('\n')
  for (const guildID of item.managementGuildIDs) {
    await bot.sendMessage(guildID, content)
  }
}

export interface FreshmanForwardBot extends Universal.Methods {
  platform?: string
  selfId: string
}

function botMatchesForward(bot: FreshmanForwardBot, item: FreshmanForwardItem) {
  const platformMatches = !item.platform || !bot.platform || bot.platform === item.platform
  return platformMatches && bot.selfId === item.botSelfID
}
