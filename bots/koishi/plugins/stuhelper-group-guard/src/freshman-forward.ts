import { h, type Universal } from 'koishi'

import type { FreshmanForwardItem } from '@stuhelper/koishi-shared'

import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'
import { formatFreshmanForwardSummary } from './admission-format'
import { validateFreshmanMaterialURL } from './freshman-material-url'

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
  const materialURL = validateFreshmanMaterialURL(item.materialURL)
  const content = [
    h.image(materialURL),
    formatFreshmanForwardSummary(item),
  ].join('\n')
  const failures: FreshmanForwardFailure[] = []
  for (const guildID of item.managementGuildIDs) {
    try {
      await bot.sendMessage(guildID, content)
    } catch (error) {
      failures.push({ guildID, error })
    }
  }
  if (failures.length) {
    throw createFreshmanForwardError(item.application.id, failures)
  }
}

export interface FreshmanForwardBot extends Universal.Methods {
  platform?: string
  selfId: string
}

function botMatchesForward(bot: FreshmanForwardBot, item: FreshmanForwardItem) {
  const botPlatform = resolveAdmissionSubjectPlatform(bot.platform)
  const platformMatches = !item.platform || !botPlatform || botPlatform === item.platform
  return platformMatches && bot.selfId === item.botSelfID
}

function createFreshmanForwardError(applicationID: string, failures: readonly FreshmanForwardFailure[]) {
  const guildIDs = failures.map((failure) => failure.guildID).join(', ')
  const errors = failures.map((failure) => normalizeForwardError(failure))
  return new AggregateError(errors, `freshman forward ${applicationID} failed for guilds: ${guildIDs}`)
}

function normalizeForwardError(failure: FreshmanForwardFailure) {
  const message = failure.error instanceof Error ? failure.error.message : String(failure.error)
  return new Error(`guild ${failure.guildID}: ${message}`)
}

interface FreshmanForwardFailure {
  readonly guildID: string
  readonly error: unknown
}
